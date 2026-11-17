package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

const (
	sddPreflightLegacyQuestionPrefix = "Gentle AI SDD preflight "
	// sddPreflightAxiomQuestionPrefix is the Axiom-branded form of the
	// preflight question prefix. Both prefixes stay accepted indefinitely:
	// installed plugins and agent instructions already written on users'
	// machines keep using the legacy prefix until the next `axiom sync`.
	sddPreflightAxiomQuestionPrefix = "Axiom SDD preflight "
	maxSDDPreflightHookPayloadBytes = 256 << 10
	maxSDDPreflightTranscriptLine   = 8 << 20
)

var sddPreflightHookSessionID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
var sddPreflightHookPhases = []string{"sdd-init", "sdd-explore", "sdd-propose", "sdd-spec", "sdd-design", "sdd-tasks", "sdd-apply", "sdd-verify", "sdd-archive", "sdd-onboard", "sdd-research"}

type sddPreflightHookOption struct {
	Label string `json:"label"`
}

type sddPreflightHookQuestion struct {
	Header      string                   `json:"header"`
	Question    string                   `json:"question"`
	Options     []sddPreflightHookOption `json:"options"`
	MultiSelect bool                     `json:"multiSelect,omitempty"`
}

type sddPreflightHookToolInput struct {
	SubagentType string `json:"subagent_type,omitempty"`
	Prompt       string `json:"prompt,omitempty"`
}

type sddPreflightHookPayload struct {
	SessionID      string
	TranscriptPath string
	CWD            string
	HookEventName  string
	ToolName       string
	AgentID        string
	AgentType      string
	ToolInput      sddPreflightHookToolInput
	ToolInputRaw   map[string]any
}

func (payload *sddPreflightHookPayload) UnmarshalJSON(raw []byte) error {
	var wire struct {
		SessionID      string          `json:"session_id"`
		TranscriptPath string          `json:"transcript_path"`
		CWD            string          `json:"cwd,omitempty"`
		HookEventName  string          `json:"hook_event_name"`
		ToolName       string          `json:"tool_name"`
		AgentID        string          `json:"agent_id,omitempty"`
		AgentType      string          `json:"agent_type,omitempty"`
		ToolInput      json.RawMessage `json:"tool_input,omitempty"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	payload.SessionID, payload.TranscriptPath, payload.CWD = wire.SessionID, wire.TranscriptPath, wire.CWD
	payload.HookEventName, payload.ToolName, payload.AgentID, payload.AgentType = wire.HookEventName, wire.ToolName, wire.AgentID, wire.AgentType
	if len(wire.ToolInput) > 0 {
		if err := json.Unmarshal(wire.ToolInput, &payload.ToolInput); err != nil {
			return err
		}
		if err := json.Unmarshal(wire.ToolInput, &payload.ToolInputRaw); err != nil {
			return err
		}
	}
	return nil
}

type sddPreflightHookSpecificOutput struct {
	HookEventName            string         `json:"hookEventName"`
	PermissionDecision       string         `json:"permissionDecision"`
	PermissionDecisionReason string         `json:"permissionDecisionReason,omitempty"`
	UpdatedInput             map[string]any `json:"updatedInput,omitempty"`
}

type sddPreflightHookOutput struct {
	HookSpecificOutput sddPreflightHookSpecificOutput `json:"hookSpecificOutput"`
}

func RunSDDPreflightHook(args []string, stdout io.Writer) error {
	return runSDDPreflightHook(args, os.Stdin, stdout)
}

// runSDDPreflightHook implements `gentle-ai sdd-preflight-hook --agent
// claude-code`, installed as a Claude Code PreToolUse(Agent) hook. Authority
// is derived at dispatch time directly from the session transcript that the
// Claude Code hook runner supplies on stdin (transcript_path and
// session_id): a model-started copy of this command cannot influence the
// real dispatch decision, because Claude Code only honors the output of the
// hook invocation it started itself, and that invocation's stdin is
// runner-supplied. There is no minted, persisted record of authority.
func runSDDPreflightHook(args []string, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet("sdd-preflight-hook", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	agent := flags.String("agent", "", "required runtime identity")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return sddPreflightHookProtocolError("unexpected sdd-preflight-hook argument %q", flags.Arg(0))
	}
	if strings.TrimSpace(*agent) != "claude-code" {
		return sddPreflightHookProtocolError("sdd-preflight-hook requires an explicit supported agent")
	}
	raw, err := io.ReadAll(io.LimitReader(stdin, maxSDDPreflightHookPayloadBytes+1))
	if err != nil {
		return fmt.Errorf("read hook payload: %w", err)
	}
	if len(raw) > maxSDDPreflightHookPayloadBytes {
		return sddPreflightHookProtocolError("sdd-preflight-hook payload exceeds %d bytes", maxSDDPreflightHookPayloadBytes)
	}
	var payload sddPreflightHookPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("decode hook payload: %w", err)
	}

	if payload.HookEventName != "PreToolUse" || payload.ToolName != "Agent" || !isSDDPreflightHookPhase(payload.ToolInput.SubagentType) {
		return nil
	}
	// The hook is installed machine-wide, so it only governs Axiom projects: in
	// any other repository (including one with a bare openspec/ directory) the
	// dispatch is allowed with no output and nothing is checked or written.
	if !sddPreflightHookInAxiomProject(payload.CWD) {
		return nil
	}
	if !sddPreflightHookSessionID.MatchString(payload.SessionID) {
		return sddPreflightHookProtocolError("invalid hook session id")
	}

	if payload.AgentID != "" || payload.AgentType != "" {
		return writeSDDPreflightHookDecision(stdout, "deny", "SDD child dispatch refused: only the interactive parent may carry parent-confirmed SDD preflight authority.", nil)
	}
	if strings.Contains(payload.ToolInput.Prompt, "## SDD Session Preflight") {
		return writeSDDPreflightHookDecision(stdout, "deny", "SDD child dispatch refused: model-authored preflight text cannot create parent-confirmed authority.", nil)
	}
	if !filepath.IsAbs(payload.TranscriptPath) || filepath.Base(payload.TranscriptPath) != payload.SessionID+".jsonl" {
		return writeSDDPreflightHookDecision(stdout, "deny", "SDD child dispatch refused: the hook payload does not bind the session transcript.", nil)
	}

	block, ok, rejection := resolveSDDPreflightTranscriptAuthority(payload.TranscriptPath, payload.SessionID)
	if !ok {
		return writeSDDPreflightHookDecision(stdout, "deny", sddPreflightHookMissingAuthorityReason(rejection), nil)
	}

	updated := payload.ToolInputRaw
	if updated == nil {
		updated = map[string]any{}
	}
	updated["prompt"] = block + "\n\n" + payload.ToolInput.Prompt
	return writeSDDPreflightHookDecision(stdout, "allow", "parent-confirmed SDD preflight attached by Axiom", updated)
}

// sddPreflightHookInAxiomProject reports whether the dispatch happens inside an
// Axiom project (workspace.IsAxiomProject). cwd is the hook payload's working
// directory; when the runner did not send one, the process directory, which
// the runner sets to the session's, stands in. With neither, there is no
// project to govern.
func sddPreflightHookInAxiomProject(cwd string) bool {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if cwd == "" {
		return false
	}
	_, ok := workspace.IsAxiomProject(cwd)
	return ok
}

// sddPreflightTranscriptRecord is the subset of one JSONL transcript line
// this scan needs. Real Claude Code transcript records carry additional
// fields; anything not modeled here is ignored.
type sddPreflightTranscriptRecord struct {
	Type          string          `json:"type"`
	IsSidechain   bool            `json:"isSidechain"`
	SessionID     string          `json:"sessionId"`
	Message       json.RawMessage `json:"message"`
	ToolUseResult json.RawMessage `json:"toolUseResult"`
}

type sddPreflightTranscriptMessage struct {
	Content []sddPreflightTranscriptContentBlock `json:"content"`
}

type sddPreflightTranscriptContentBlock struct {
	Type      string          `json:"type"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// resolveSDDPreflightTranscriptAuthority scans the session transcript at
// path for a corroborated canonical SDD preflight: an assistant
// AskUserQuestion tool_use asking the canonical grouped questions, matched
// by tool_use id to a later, non-error user tool_result carrying
// toolUseResult.answers for that same session (isSidechain false). The last
// corroborated pair in file order wins. When none is valid, rejection is the
// reason the last recognized preflight was refused (nil when there was none),
// so the denial can say what to correct. Malformed or unparsable lines are
// skipped, never fatal; a line longer than maxSDDPreflightTranscriptLine has
// its remainder discarded without being buffered, and scanning continues
// with the next line.
func resolveSDDPreflightTranscriptAuthority(path, sessionID string) (block string, found bool, rejection error) {
	file, err := os.Open(path)
	if err != nil {
		return "", false, nil
	}
	defer file.Close()

	pending := map[string][]sddPreflightHookQuestion{}

	reader := bufio.NewReader(file)
	for {
		line, ok, done := readSDDPreflightTranscriptLine(reader)
		if ok && len(strings.TrimSpace(string(line))) > 0 {
			var record sddPreflightTranscriptRecord
			if err := json.Unmarshal(line, &record); err == nil && !record.IsSidechain && record.SessionID == sessionID && len(record.Message) > 0 {
				var message sddPreflightTranscriptMessage
				if err := json.Unmarshal(record.Message, &message); err == nil {
					switch record.Type {
					case "assistant":
						for _, part := range message.Content {
							if part.Type != "tool_use" || part.Name != "AskUserQuestion" || part.ID == "" {
								continue
							}
							var wrapped struct {
								Questions []sddPreflightHookQuestion `json:"questions"`
							}
							if err := json.Unmarshal(part.Input, &wrapped); err != nil {
								continue
							}
							pending[part.ID] = wrapped.Questions
						}
					case "user":
						for _, part := range message.Content {
							if part.Type != "tool_result" || part.ToolUseID == "" || part.IsError {
								continue
							}
							questions, ok := pending[part.ToolUseID]
							if !ok || len(record.ToolUseResult) == 0 {
								continue
							}
							var toolUseResult struct {
								Answers json.RawMessage `json:"answers"`
							}
							if err := json.Unmarshal(record.ToolUseResult, &toolUseResult); err != nil {
								continue
							}
							candidate, recognized, err := resolveSDDPreflightHookBlock(questions, toolUseResult.Answers)
							if !recognized {
								continue
							}
							if err != nil {
								rejection = err
								continue
							}
							if !validSDDPreflightHookBlock(candidate) {
								continue
							}
							block, found = candidate, true
						}
					}
				}
			}
		}
		if done {
			break
		}
	}
	if found {
		rejection = nil
	}
	return block, found, rejection
}

// sddPreflightHookMissingAuthorityReason is the denial text when no valid
// preflight authorizes the dispatch. When the last recognized preflight was
// refused, the text names that concrete cause so it can be corrected.
func sddPreflightHookMissingAuthorityReason(rejection error) string {
	const missing = "SDD child dispatch refused: parent-confirmed SDD preflight is missing, invalid, or uncorroborated."
	if rejection == nil {
		return missing + " Ask the canonical grouped preflight with AskUserQuestion and stop before retrying."
	}
	return missing + " The last SDD preflight in this session was rejected: " + rejection.Error() + ". Fix that, ask the canonical grouped preflight again with AskUserQuestion and stop before retrying."
}

// readSDDPreflightTranscriptLine reads one newline- or EOF-terminated line
// from r without ever buffering more than maxSDDPreflightTranscriptLine
// bytes of it. If the line exceeds that bound, ok is false and the
// remainder of the oversized line is discarded (read and dropped, never
// accumulated) so r is left positioned at the start of the next line. done
// is true once there is no further data to read; when done is true, line
// and ok may still carry the final line of the file.
func readSDDPreflightTranscriptLine(r *bufio.Reader) (line []byte, ok bool, done bool) {
	var buf []byte
	overflow := false
	for {
		chunk, err := r.ReadSlice('\n')
		if len(chunk) > 0 && !overflow {
			if len(buf)+len(chunk) > maxSDDPreflightTranscriptLine {
				overflow = true
				buf = nil
			} else {
				buf = append(buf, chunk...)
			}
		}
		switch err {
		case nil:
			if overflow {
				return nil, false, false
			}
			return bytes.TrimSuffix(buf, []byte("\n")), true, false
		case bufio.ErrBufferFull:
			continue
		case io.EOF:
			if overflow || len(buf) == 0 {
				return nil, false, true
			}
			return buf, true, true
		default:
			return nil, false, true
		}
	}
}

// sddPreflightQuestionPrefixes lists every accepted question prefix.
func sddPreflightQuestionPrefixes() []string {
	return []string{sddPreflightLegacyQuestionPrefix, sddPreflightAxiomQuestionPrefix}
}

// hasSDDPreflightQuestionPrefix reports whether the question starts with any
// accepted preflight prefix.
func hasSDDPreflightQuestionPrefix(question string) bool {
	for _, prefix := range sddPreflightQuestionPrefixes() {
		if strings.HasPrefix(question, prefix) {
			return true
		}
	}
	return false
}

// hasSDDPreflightQuestionMarker reports whether the question starts with the
// `N/3:` marker of the given zero-based index under any accepted prefix.
func hasSDDPreflightQuestionMarker(question string, index int) bool {
	for _, prefix := range sddPreflightQuestionPrefixes() {
		if strings.HasPrefix(question, fmt.Sprintf("%s%d/3:", prefix, index+1)) {
			return true
		}
	}
	return false
}

// sddPreflightChoice binds one canonical option label to its internal token.
type sddPreflightChoice struct {
	Label string
	Token string
}

// sddPreflightGroup is one canonical preflight question. Its position in
// sddPreflightGroups is its marker: the first group is asked as `1/3:`, and
// so on.
type sddPreflightGroup struct {
	Name    string
	Choices []sddPreflightChoice
}

// sddPreflightGroups is the canonical preflight. The hook binds every
// question by its marker and every answer by its label, so neither the order
// of the questions nor the order of their options changes the outcome.
var sddPreflightGroups = []sddPreflightGroup{
	{Name: "Pace", Choices: []sddPreflightChoice{{"Interactive", "interactive"}, {"Automatic", "auto"}}},
	{Name: "Artifacts", Choices: []sddPreflightChoice{{"OpenSpec", "openspec"}, {"Engram", "engram"}, {"Both", "hybrid"}}},
	{Name: "PR strategy", Choices: []sddPreflightChoice{{"Ask me", "ask-on-risk"}, {"Single PR", "single-pr"}, {"Auto", "auto-chain"}}},
}

// sddPreflightRecommendedSuffix is the suffix the AskUserQuestion guidance
// asks models to append to the label of the option they recommend.
const sddPreflightRecommendedSuffix = "(recommended)"

// normalizeSDDPreflightLabel folds the presentation of a label: surrounding
// whitespace, letter case and a trailing "(Recommended)" never change which
// canonical option it names.
func normalizeSDDPreflightLabel(label string) string {
	normalized := strings.ToLower(strings.TrimSpace(label))
	return strings.TrimSpace(strings.TrimSuffix(normalized, sddPreflightRecommendedSuffix))
}

// choice returns the index of the canonical choice the label names, or -1.
func (group sddPreflightGroup) choice(label string) int {
	normalized := normalizeSDDPreflightLabel(label)
	for index, choice := range group.Choices {
		if normalized == strings.ToLower(choice.Label) {
			return index
		}
	}
	return -1
}

func (group sddPreflightGroup) labels() string {
	labels := make([]string, len(group.Choices))
	for index, choice := range group.Choices {
		labels[index] = choice.Label
	}
	return strings.Join(labels, ", ")
}

// sddPreflightQuestionMarkerIndex returns the zero-based group index of the
// `N/3:` marker the question starts with under any accepted prefix, or -1.
func sddPreflightQuestionMarkerIndex(question string) int {
	for index := range sddPreflightGroups {
		if hasSDDPreflightQuestionMarker(question, index) {
			return index
		}
	}
	return -1
}

// resolveSDDPreflightHookBlock derives the canonical preflight block from one
// grouped AskUserQuestion call and its answers. recognized is false when no
// question carries a preflight prefix. Each question is bound to its group by
// its marker and each option and answer to a canonical choice by its label,
// so presentation order never matters; every error names the question and
// what to correct.
func resolveSDDPreflightHookBlock(questions []sddPreflightHookQuestion, answersRaw json.RawMessage) (string, bool, error) {
	recognized := false
	for _, question := range questions {
		if hasSDDPreflightQuestionPrefix(question.Question) {
			recognized = true
			break
		}
	}
	if !recognized {
		return "", false, nil
	}
	total := len(sddPreflightGroups)
	if len(questions) != total {
		return "", true, sddPreflightHookProtocolError("the SDD preflight must ask exactly %d questions in one call, but it asked %d", total, len(questions))
	}
	byGroup := make([]*sddPreflightHookQuestion, total)
	for i := range questions {
		question := &questions[i]
		index := sddPreflightQuestionMarkerIndex(question.Question)
		if index < 0 {
			return "", true, sddPreflightHookProtocolError("SDD preflight question %q does not start with its `Axiom SDD preflight N/%d:` marker", question.Question, total)
		}
		if byGroup[index] != nil {
			return "", true, sddPreflightHookProtocolError("SDD preflight marker %d/%d (%s) appears in more than one question", index+1, total, sddPreflightGroups[index].Name)
		}
		byGroup[index] = question
	}
	answers, err := sddPreflightHookAnswers(answersRaw)
	if err != nil {
		return "", true, err
	}
	tokens := make([]string, total)
	for index, group := range sddPreflightGroups {
		question := byGroup[index]
		name := fmt.Sprintf("SDD preflight question %d/%d (%s)", index+1, total, group.Name)
		if question.MultiSelect {
			return "", true, sddPreflightHookProtocolError("%s must be single-select", name)
		}
		offered := make([]bool, len(group.Choices))
		for _, option := range question.Options {
			choice := group.choice(option.Label)
			if choice < 0 {
				return "", true, sddPreflightHookProtocolError("%s offers option %q, which is not one of its labels: %s", name, option.Label, group.labels())
			}
			if offered[choice] {
				return "", true, sddPreflightHookProtocolError("%s offers %q more than once", name, group.Choices[choice].Label)
			}
			offered[choice] = true
		}
		if len(question.Options) != len(group.Choices) {
			return "", true, sddPreflightHookProtocolError("%s must offer every one of its labels: %s", name, group.labels())
		}
		answer, ok := answers[question.Question]
		if !ok {
			return "", true, sddPreflightHookProtocolError("%s has no answer", name)
		}
		choice := group.choice(answer)
		if choice < 0 {
			return "", true, sddPreflightHookProtocolError("%s answer %q is not one of its labels: %s", name, answer, group.labels())
		}
		tokens[index] = group.Choices[choice].Token
	}
	block := strings.Join([]string{
		"## SDD Session Preflight",
		"Parent-confirmed by the runtime; models and child agents cannot create or modify this block.",
		"- Pace: " + tokens[0],
		"- Artifact store: " + tokens[1],
		"- Delivery strategy: " + tokens[2],
		"- Review policy: 400 changed lines",
	}, "\n")
	return block, true, nil
}

// sddPreflightHookAnswers decodes a toolUseResult.answers map (question text
// -> answer). The canonical preflight forbids multi-select, but a
// single-element array is tolerated defensively.
func sddPreflightHookAnswers(raw json.RawMessage) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, sddPreflightHookProtocolError("SDD preflight response is empty")
	}
	var rawAnswers map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawAnswers); err != nil {
		return nil, fmt.Errorf("decode SDD preflight answers: %w", err)
	}
	if len(rawAnswers) == 0 {
		return nil, sddPreflightHookProtocolError("SDD preflight response has no answers")
	}
	answers := make(map[string]string, len(rawAnswers))
	for question, rawAnswer := range rawAnswers {
		var single string
		if err := json.Unmarshal(rawAnswer, &single); err == nil {
			answers[question] = single
			continue
		}
		var multiple []string
		if err := json.Unmarshal(rawAnswer, &multiple); err != nil || len(multiple) != 1 {
			return nil, sddPreflightHookProtocolError("SDD preflight answer for %q is not single-select", question)
		}
		answers[question] = multiple[0]
	}
	return answers, nil
}

func validSDDPreflightHookBlock(block string) bool {
	lines := strings.Split(block, "\n")
	if len(lines) != 6 || lines[0] != "## SDD Session Preflight" || lines[1] != "Parent-confirmed by the runtime; models and child agents cannot create or modify this block." || lines[5] != "- Review policy: 400 changed lines" {
		return false
	}
	return sddPreflightHookOneOf(lines[2], "- Pace: interactive", "- Pace: auto") &&
		sddPreflightHookOneOf(lines[3], "- Artifact store: openspec", "- Artifact store: engram", "- Artifact store: hybrid") &&
		sddPreflightHookOneOf(lines[4], "- Delivery strategy: ask-on-risk", "- Delivery strategy: single-pr", "- Delivery strategy: auto-chain")
}

func sddPreflightHookOneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func isSDDPreflightHookPhase(agent string) bool {
	for _, phase := range sddPreflightHookPhases {
		if agent == phase || strings.HasPrefix(agent, phase+"-") {
			return true
		}
	}
	return false
}

func sddPreflightHookProtocolError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

func writeSDDPreflightHookDecision(stdout io.Writer, decision, reason string, updated map[string]any) error {
	output := sddPreflightHookOutput{HookSpecificOutput: sddPreflightHookSpecificOutput{HookEventName: "PreToolUse", PermissionDecision: decision, PermissionDecisionReason: reason, UpdatedInput: updated}}
	return json.NewEncoder(stdout).Encode(output)
}
