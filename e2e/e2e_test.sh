#!/usr/bin/env bash
# e2e_test.sh — End-to-end tests for Axiom installer
#
# Test tiers (controlled by environment variables):
#   (default)            Tier 1: binary existence + dry-run tests (fast, no side-effects)
#   RUN_FULL_E2E=1       Tier 2: full install tests (writes to filesystem)
#   RUN_BACKUP_TESTS=1   Tier 3: backup/restore tests
#
# Usage inside Docker:
#   ./e2e_test.sh                         # Tier 1 only
#   RUN_FULL_E2E=1 ./e2e_test.sh          # Tier 1 + 2
#   RUN_BACKUP_TESTS=1 ./e2e_test.sh      # Tier 1 + 3
#   RUN_FULL_E2E=1 RUN_BACKUP_TESTS=1 ./e2e_test.sh  # All tiers
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "$SCRIPT_DIR/lib.sh"

# ---------------------------------------------------------------------------
# Resolve binary
# ---------------------------------------------------------------------------
BINARY="$(resolve_binary)"
if [ -z "$BINARY" ]; then
    echo "ERROR: axiom binary not found. Build it first."
    exit 1
fi
log_info "Using binary: $BINARY"

# Side-effect E2E exercises install/injection behavior. Keep it deterministic by
# satisfying the installer's "engram already exists on PATH" branch unless a
# maintainer explicitly opts into the live GitHub release download path.
if [ "${RUN_FULL_E2E:-0}" = "1" ] || [ "${RUN_BACKUP_TESTS:-0}" = "1" ]; then
    setup_fake_engram_binary
fi

# ===========================================================================
# TIER 1 — Basic binary & dry-run tests (always run)
# ===========================================================================

# --- Category 1a: Binary basics ---

test_binary_exists() {
    log_test "Binary exists and is executable"

    if [ -x "$(command -v "$BINARY")" ] || [ -x "$BINARY" ]; then
        log_pass "Binary is executable"
    else
        log_fail "Binary not found or not executable"
    fi
}

test_binary_runs() {
    log_test "Binary runs without panic"

    if output=$($BINARY install --dry-run 2>&1); then
        log_pass "Binary exited cleanly with --dry-run"
    else
        if echo "$output" | grep -qi "panic"; then
            log_fail "Binary panicked: $output"
        else
            log_pass "Binary exited with non-zero (no panic)"
        fi
    fi
}

test_version_command() {
    log_test "Version command works"

    output=$($BINARY version 2>&1) || true

    if echo "$output" | grep -q "axiom"; then
        log_pass "Version command returns binary name"
    else
        log_fail "Version command failed: $output"
    fi
}

# --- Category 1b: Dry-run output format ---

test_dry_run_output_format() {
    log_test "Dry-run output contains expected sections"

    output=$($BINARY install --dry-run 2>&1) || true

    assert_output_contains "$output" "dry-run" "Output contains 'dry-run' marker"
    assert_output_contains "$output" "Agents:" "Output contains 'Agents:' header"
    assert_output_contains "$output" "Components order:" "Output contains 'Components order:' header"
    assert_output_contains "$output" "Platform decision:" "Output contains 'Platform decision:' header"
}

test_dry_run_platform_detection() {
    log_test "Dry-run shows platform decision"

    output=$($BINARY install --dry-run 2>&1) || true

    assert_output_contains "$output" "Platform decision" "Platform decision present in dry-run"
}

test_dry_run_detects_linux() {
    log_test "Dry-run detects Linux OS"

    # This test is only meaningful inside the Docker container (Linux).
    # Skip gracefully on macOS/other to avoid killing the test run.
    if [[ "$(uname -s)" != "Linux" ]]; then
        log_skip "Not running on Linux — platform detection test skipped"
        return 0
    fi

    output=$($BINARY install --dry-run 2>&1) || true

    assert_output_contains "$output" "os=linux" "Platform detected as Linux"
}

# --- Category 1c: Agent flag ---

test_dry_run_agent_claude_code() {
    log_test "Dry-run with --agent claude-code"

    output=$($BINARY install --agent claude-code --dry-run 2>&1) || true

    assert_output_contains "$output" "claude-code" "Dry-run output shows claude-code agent"
}

test_dry_run_agent_opencode() {
    log_test "Dry-run with --agent opencode"

    output=$($BINARY install --agent opencode --dry-run 2>&1) || true

    assert_output_contains "$output" "opencode" "Dry-run output shows opencode agent"
}

test_dry_run_agent_both() {
    log_test "Dry-run with both agents"

    output=$($BINARY install --agent claude-code --agent opencode --dry-run 2>&1) || true

    assert_output_contains "$output" "claude-code" "Both agents: shows claude-code"
    assert_output_contains "$output" "opencode" "Both agents: shows opencode"
}

test_dry_run_agent_csv() {
    log_test "Dry-run with --agent as CSV"

    output=$($BINARY install --agent claude-code,opencode --dry-run 2>&1) || true

    assert_output_contains "$output" "claude-code" "CSV agents: shows claude-code"
    assert_output_contains "$output" "opencode" "CSV agents: shows opencode"
}

# --- Category 1d: Default component set (no --component) ---

test_dry_run_default_persona_before_sdd() {
    log_test "Dry-run: persona appears before engram and sdd in component order"

    output=$($BINARY install --agent opencode --dry-run 2>&1) || true

    local components_line
    components_line=$(echo "$output" | grep "Components order:")

    # Verify all are present
    assert_output_contains "$components_line" "persona" "Default set has persona"
    assert_output_contains "$components_line" "engram" "Default set has engram"
    assert_output_contains "$components_line" "sdd" "Default set has sdd"

    # Verify ordering: persona before engram, persona before sdd
    # Extract the order string and check persona comes first
    local order_str
    order_str=$(echo "$components_line" | sed 's/.*Components order: *//')

    local persona_idx engram_idx sdd_idx
    persona_idx=$(echo "$order_str" | tr ',' '\n' | grep -n '^persona$' | cut -d: -f1)
    engram_idx=$(echo "$order_str" | tr ',' '\n' | grep -n '^engram$' | cut -d: -f1)
    sdd_idx=$(echo "$order_str" | tr ',' '\n' | grep -n '^sdd$' | cut -d: -f1)

    if [ -n "$persona_idx" ] && [ -n "$engram_idx" ] && [ "$persona_idx" -lt "$engram_idx" ]; then
        log_pass "Persona ($persona_idx) before engram ($engram_idx)"
    else
        log_fail "Persona must appear before engram in component order: $order_str"
    fi

    if [ -n "$persona_idx" ] && [ -n "$sdd_idx" ] && [ "$persona_idx" -lt "$sdd_idx" ]; then
        log_pass "Persona ($persona_idx) before sdd ($sdd_idx)"
    else
        log_fail "Persona must appear before sdd in component order: $order_str"
    fi
}

test_dry_run_explicit_components() {
    log_test "Dry-run with explicit --component flags"

    output=$($BINARY install --agent claude-code --component engram --component sdd --component skills --dry-run 2>&1) || true

    local components_line
    components_line=$(echo "$output" | grep "Components order:")
    assert_output_contains "$components_line" "engram" "Custom + explicit components includes engram"
    assert_output_contains "$components_line" "sdd" "Custom + explicit components includes sdd"
    assert_output_contains "$components_line" "skills" "Custom + explicit components includes skills"
    assert_output_not_contains "$components_line" "persona" "Explicit components exclude persona"
    assert_output_not_contains "$components_line" "context7" "Custom + explicit components excludes context7"
}

# --- Category 1f: Individual component flags ---

test_dry_run_component_engram() {
    log_test "Dry-run with --component engram"
    output=$($BINARY install --agent claude-code --component engram --dry-run 2>&1) || true
    assert_output_contains "$output" "engram" "Shows engram component"
}

test_dry_run_component_sdd() {
    log_test "Dry-run with --component sdd"
    output=$($BINARY install --agent claude-code --component sdd --dry-run 2>&1) || true
    assert_output_contains "$output" "sdd" "Shows sdd component"
}

test_dry_run_component_skills() {
    log_test "Dry-run with --component skills"
    output=$($BINARY install --agent claude-code --component skills --dry-run 2>&1) || true
    assert_output_contains "$output" "skills" "Shows skills component"
}

test_dry_run_component_context7() {
    log_test "Dry-run with --component context7"
    output=$($BINARY install --agent claude-code --component context7 --dry-run 2>&1) || true
    assert_output_contains "$output" "context7" "Shows context7 component"
}

test_dry_run_component_persona() {
    log_test "Dry-run with --component persona"
    output=$($BINARY install --agent claude-code --component persona --dry-run 2>&1) || true
    assert_output_contains "$output" "persona" "Shows persona component"
}

test_dry_run_component_permissions() {
    log_test "Dry-run with --component permissions"
    output=$($BINARY install --agent opencode --component permissions --dry-run 2>&1) || true
    assert_output_contains "$output" "permissions" "Shows permissions component"
}

# --- Category 1f2: SDD mode flag ---

test_dry_run_sdd_mode_multi() {
    log_test "Dry-run with --sdd-mode multi"

    output=$($BINARY install --agent opencode --sdd-mode multi --dry-run 2>&1) || true

    assert_output_contains "$output" "opencode" "Shows opencode agent"
    assert_output_contains "$output" "sdd-mode: multi\|SDDMode: multi\|sdd_mode.*multi\|multi" "Shows multi mode"
}

test_dry_run_sdd_mode_single() {
    log_test "Dry-run with --sdd-mode single"

    output=$($BINARY install --agent opencode --sdd-mode single --dry-run 2>&1) || true

    assert_output_contains "$output" "opencode" "Shows opencode agent"
}

test_dry_run_sdd_mode_invalid_rejected() {
    log_test "Invalid --sdd-mode is rejected"

    if $BINARY install --agent opencode --sdd-mode turbo --dry-run 2>&1; then
        log_fail "Invalid sdd-mode should have been rejected"
    else
        log_pass "Invalid sdd-mode correctly rejected"
    fi
}

# --- Category 1g: Invalid input rejection ---

test_invalid_component_rejected() {
    log_test "Invalid component is rejected"

    if $BINARY install --component fakecomp --dry-run 2>&1; then
        log_fail "Invalid component should have been rejected"
    else
        log_pass "Invalid component correctly rejected"
    fi
}

test_unknown_command_rejected() {
    log_test "Unknown command is rejected"

    if $BINARY foobar 2>&1; then
        log_fail "Unknown command should have been rejected"
    else
        log_pass "Unknown command correctly rejected"
    fi
}

# ===========================================================================
# TIER 2 — Full install tests (require RUN_FULL_E2E=1)
# ===========================================================================

# --- Category 2: Claude Code component injection ---

test_cc_engram_injection() {
    log_test "Claude Code: engram injection (MCP + CLAUDE.md)"
    cleanup_test_env

    if $BINARY install --agent claude-code --component engram 2>&1; then
        # User-scope MCP registry
        local registry="$HOME/.claude.json"
        assert_file_exists "$registry" "Claude user MCP registry"
        assert_file_contains "$registry" '"mcpServers"' "Registry has mcpServers"
        assert_file_contains "$registry" '"engram"' "Registry has Engram server"
        assert_valid_json "$registry" "Claude user MCP registry is valid JSON"
        assert_file_not_exists "$HOME/.claude/mcp/engram.json" "legacy Engram MCP file is not written"

        # CLAUDE.md section
        assert_file_exists "$HOME/.claude/CLAUDE.md" "CLAUDE.md exists"
        assert_file_contains_marker "$HOME/.claude/CLAUDE.md" "engram-protocol" "CLAUDE.md has engram-protocol section marker"
        assert_file_contains "$HOME/.claude/CLAUDE.md" "mem_save" "CLAUDE.md has real Engram content (mem_save)"
        assert_file_size_min "$HOME/.claude/CLAUDE.md" 500 "CLAUDE.md has substantial content"
    else
        log_fail "engram install command failed"
    fi
}

test_cc_sdd_injection() {
    log_test "Claude Code: SDD injection (CLAUDE.md + native sub-agents)"
    cleanup_test_env

    if $BINARY install --agent claude-code --component sdd 2>&1; then
        assert_file_exists "$HOME/.claude/CLAUDE.md" "CLAUDE.md exists"
        assert_file_contains_marker "$HOME/.claude/CLAUDE.md" "sdd-orchestrator" "CLAUDE.md has SDD section marker"
        assert_file_contains "$HOME/.claude/CLAUDE.md" "sub-agent\|dependency\|orchestrator" "CLAUDE.md has real SDD content"
        assert_file_size_min "$HOME/.claude/CLAUDE.md" 500 "CLAUDE.md SDD section is substantial"

        for phase in sdd-init sdd-explore sdd-research sdd-propose sdd-spec sdd-design sdd-tasks sdd-apply sdd-verify sdd-archive sdd-onboard; do
            assert_file_exists "$HOME/.claude/agents/${phase}.md" "Claude native sub-agent exists: ${phase}"
            assert_file_size_min "$HOME/.claude/agents/${phase}.md" 200 "Claude native sub-agent is substantial: ${phase}"
        done

        assert_file_contains "$HOME/.claude/agents/sdd-design.md" "model: opus" "Claude design sub-agent uses balanced Opus assignment"
        assert_file_contains "$HOME/.claude/agents/sdd-spec.md" "model: sonnet" "Claude spec sub-agent uses balanced Sonnet assignment"
        assert_file_contains "$HOME/.claude/agents/sdd-archive.md" "model: haiku" "Claude archive sub-agent uses balanced Haiku assignment"

        assert_file_contains "$HOME/.claude/agents/sdd-explore.md" "tools:" "Claude explore sub-agent declares tool scope"
        assert_file_contains "$HOME/.claude/agents/sdd-explore.md" "WebFetch" "Claude explore sub-agent includes WebFetch"
        assert_file_contains "$HOME/.claude/agents/sdd-explore.md" "WebSearch" "Claude explore sub-agent includes WebSearch"
        assert_file_contains "$HOME/.claude/agents/sdd-explore.md" "mcp__plugin_engram_engram__mem_save" "Claude explore sub-agent includes Engram save"

        assert_file_contains "$HOME/.claude/agents/sdd-apply.md" "tools:" "Claude apply sub-agent declares tool scope"
        assert_file_contains "$HOME/.claude/agents/sdd-apply.md" "Read" "Claude apply sub-agent includes Read"
        assert_file_contains "$HOME/.claude/agents/sdd-apply.md" "Edit" "Claude apply sub-agent includes Edit"
        assert_file_contains "$HOME/.claude/agents/sdd-apply.md" "Write" "Claude apply sub-agent includes Write"
        assert_file_contains "$HOME/.claude/agents/sdd-apply.md" "Bash" "Claude apply sub-agent includes Bash"
        assert_file_contains "$HOME/.claude/agents/sdd-apply.md" "mcp__plugin_engram_engram__mem_search" "Claude apply sub-agent includes Engram search"
        assert_file_contains "$HOME/.claude/agents/sdd-apply.md" "mcp__plugin_engram_engram__mem_get_observation" "Claude apply sub-agent includes Engram read"
        assert_file_contains "$HOME/.claude/agents/sdd-apply.md" "mcp__plugin_engram_engram__mem_save" "Claude apply sub-agent includes Engram save"
        assert_file_contains "$HOME/.claude/agents/sdd-apply.md" "mcp__plugin_engram_engram__mem_update" "Claude apply sub-agent includes Engram update"

        assert_file_contains "$HOME/.claude/agents/sdd-verify.md" "tools:" "Claude verify sub-agent declares tool scope"
        assert_file_contains "$HOME/.claude/agents/sdd-verify.md" "Read" "Claude verify sub-agent includes Read"
        assert_file_contains "$HOME/.claude/agents/sdd-verify.md" "Bash" "Claude verify sub-agent includes Bash"
        assert_file_contains "$HOME/.claude/agents/sdd-verify.md" "mcp__plugin_engram_engram__mem_search" "Claude verify sub-agent includes Engram search"
        assert_file_contains "$HOME/.claude/agents/sdd-verify.md" "mcp__plugin_engram_engram__mem_get_observation" "Claude verify sub-agent includes Engram read"
        assert_file_contains "$HOME/.claude/agents/sdd-verify.md" "mcp__plugin_engram_engram__mem_save" "Claude verify sub-agent includes Engram save"
    else
        log_fail "SDD install command failed"
    fi
}

test_cc_skills_full() {
    log_test "Claude Code: skills injection (default skills)"
    cleanup_test_env

    if $BINARY install --agent claude-code --component skills 2>&1; then
        local skills_dir="$HOME/.claude/skills"
        assert_dir_exists "$skills_dir" "Claude skills directory"

        # #3554: skills alone no longer pulls sdd. the skill catalog
        # is 13 foundation + judgment-day = 14 files; the 11 sdd-* phase
        # skills come only from the SDD component (not selected here).
        assert_file_count "$skills_dir" "SKILL.md" 8 "Skills alone: 8 skill files"
        assert_file_not_exists "$skills_dir/sdd-init/SKILL.md" "sdd-init NOT installed by skills alone"

        # Verify foundation skills exist
        assert_file_exists "$skills_dir/go-testing/SKILL.md" "go-testing SKILL.md"
        assert_file_exists "$skills_dir/skill-creator/SKILL.md" "skill-creator SKILL.md"
        assert_file_not_exists "$skills_dir/issue-creation/SKILL.md" "issue-creation NOT installed by default"
        assert_file_exists "$skills_dir/skill-registry/SKILL.md" "skill-registry SKILL.md"

        # Real content check
        assert_file_size_min "$skills_dir/go-testing/SKILL.md" 200 "go-testing skill has real content"
        assert_file_size_min "$skills_dir/skill-creator/SKILL.md" 200 "skill-creator skill has real content"
        assert_file_size_min "$skills_dir/skill-registry/SKILL.md" 200 "skill-registry skill has real content"
    else
        log_fail "skills (full) install command failed"
    fi
}

test_cc_custom_skills_with_flag() {
    log_test "Claude Code: explicit --skills flag installs specified skills"
    cleanup_test_env

    if $BINARY install --agent claude-code --component skills --skills go-testing,issue-creation 2>&1; then
        local skills_dir="$HOME/.claude/skills"
        assert_dir_exists "$skills_dir" "Claude skills directory"

        # The explicitly requested skills must be present. issue-creation is a
        # contributor skill that the default set installs, but `--skills` resolution
        # is exactly the escape hatch that still installs it.
        assert_file_exists "$skills_dir/go-testing/SKILL.md" "go-testing SKILL.md"
        assert_file_exists "$skills_dir/issue-creation/SKILL.md" "issue-creation SKILL.md"

        # #3554: skills has no hard dependency on sdd anymore. Total = just
        # the 2 explicitly requested skills.
        assert_file_count "$skills_dir" "SKILL.md" 2 "Custom + explicit skills: 2 files, no sdd"

        # SDD skills are NOT present (no auto-dependency)
        assert_file_not_exists "$skills_dir/sdd-init/SKILL.md" "sdd-init NOT installed (skills has no sdd dependency)"
    else
        log_fail "custom + skills flag install command failed"
    fi
}

test_cc_custom_sdd_plus_skills() {
    log_test "Claude Code: SDD + skills with explicit --skills flag"
    cleanup_test_env

    if $BINARY install --agent claude-code --component engram --component sdd --component skills --skills go-testing,issue-creation 2>&1; then
        local skills_dir="$HOME/.claude/skills"
        assert_dir_exists "$skills_dir" "Claude skills directory"

        # #3554 regression guard: skills and sdd are independent components
        # now (no hard dependency either way). When both are explicitly
        # selected, SDD writes its own 12 skills exactly once (the skills
        # injector skips sdd-* IDs to avoid a duplicate write) and the skills
        # component writes only the explicitly requested ones — no clobbering.
        assert_file_exists "$skills_dir/sdd-init/SKILL.md" "sdd-init SKILL.md (from SDD component)"

        # Skills component installs only the explicitly requested ones
        assert_file_exists "$skills_dir/go-testing/SKILL.md" "go-testing SKILL.md (from --skills flag)"
        assert_file_exists "$skills_dir/issue-creation/SKILL.md" "issue-creation SKILL.md (from --skills flag)"

        # Total: 12 SDD/orchestration skills + 2 explicit skills = 14, written once each.
        assert_file_count "$skills_dir" "SKILL.md" 14 "SDD + explicit skills: 14 skill files total, no duplicates"
    else
        log_fail "custom + SDD + skills install command failed"
    fi
}

test_cc_context7_injection() {
    log_test "Claude Code: context7 injection (~/.claude.json user MCP registry)"
    cleanup_test_env

    if $BINARY install --agent claude-code --component context7 2>&1; then
        # Claude Code only reads user-scope MCP servers from ~/.claude.json;
        # the settings.json mcpServers block earlier versions wrote is inert
        # and no longer written (issue #1868, PR #1909).
        local registry="$HOME/.claude.json"
        assert_file_exists "$registry" "Claude user MCP registry (~/.claude.json)"
        assert_file_contains "$registry" '"mcpServers"' "user registry has mcpServers key"
        assert_file_contains "$registry" '"context7"' "user registry has context7 server"
        assert_file_contains "$registry" 'context7-mcp' "user registry points to context7-mcp"
        assert_valid_json "$registry" "user registry is valid JSON"
        assert_file_not_exists "$HOME/.claude/mcp/context7.json" "legacy context7 MCP file is not written"
    else
        log_fail "context7 install command failed"
    fi
}

test_cc_permissions_injection() {
    log_test "Claude Code: permissions injection"
    cleanup_test_env

    if $BINARY install --agent claude-code --component permissions 2>&1; then
        local settings="$HOME/.claude/settings.json"
        assert_file_exists "$settings" "Claude settings.json"
        assert_file_contains "$settings" '"permissions"' "Has permissions key"
        assert_file_contains "$settings" '"deny"' "Has deny list"
        assert_valid_json "$settings" "settings.json is valid JSON"
    else
        log_fail "permissions install command failed"
    fi
}

# --- Category 3: OpenCode component injection ---

test_oc_engram_injection() {
    log_test "OpenCode: engram injection (opencode.json)"
    cleanup_test_env

    if $BINARY install --agent opencode --component engram 2>&1; then
        local settings="$HOME/.config/opencode/opencode.json"
        local agents_md="$HOME/.config/opencode/AGENTS.md"
        assert_file_exists "$settings" "OpenCode opencode.json"
        assert_file_contains "$settings" '"mcp"' "Has mcp key"
        assert_file_contains "$settings" '"engram"' "Has engram MCP entry"
        assert_file_contains "$settings" '"command"' "Has command key"
        assert_file_contains "$settings" '"type": "local"' "Engram uses local MCP type"
        assert_valid_json "$settings" "opencode.json is valid JSON"

        # Fallback safety: AGENTS.md must include engram protocol section.
        assert_file_exists "$agents_md" "OpenCode AGENTS.md"
        assert_file_contains_marker "$agents_md" "engram-protocol" "AGENTS.md has engram-protocol section"
        assert_file_contains "$agents_md" 'mem_save' "AGENTS.md has memory protocol content"
    else
        log_fail "OpenCode engram install command failed"
    fi
}

test_oc_sdd_injection() {
    log_test "OpenCode: SDD injection (commands + skills)"
    cleanup_test_env

    if $BINARY install --agent opencode --component sdd 2>&1; then
        local commands_dir="$HOME/.config/opencode/commands"
        local skill_dir="$HOME/.config/opencode/skills"

        # Command files (11 SDD commands from embedded assets)
        assert_dir_exists "$commands_dir" "OpenCode commands directory"
        assert_file_count "$commands_dir" "sdd-*.md" 11 "All 11 SDD command files"

        # Validate command file content
        assert_file_exists "$commands_dir/sdd-init.md" "sdd-init command file"
        assert_file_contains "$commands_dir/sdd-init.md" "sdd" "sdd-init command has SDD content"

        # SDD phases and judgment-day (12 files). _shared is support-only.
        assert_dir_exists "$skill_dir" "OpenCode skill directory"
        assert_file_count "$skill_dir" "SKILL.md" 12 "All 12 SDD and orchestration skill files"

        # Validate skill file content
        assert_file_exists "$skill_dir/sdd-init/SKILL.md" "sdd-init SKILL.md"
        assert_file_size_min "$skill_dir/sdd-init/SKILL.md" 100 "sdd-init skill has real content"
    else
        log_fail "OpenCode SDD install command failed"
    fi
}

test_oc_skills_full() {
    log_test "OpenCode: skills injection (default skills)"
    cleanup_test_env

    # #3554: skills alone no longer pulls sdd. 13 foundation + judgment-day = 14.
    if $BINARY install --agent opencode --component skills 2>&1; then
        local skill_dir="$HOME/.config/opencode/skills"
        assert_dir_exists "$skill_dir" "OpenCode skill directory"
        assert_file_count "$skill_dir" "SKILL.md" 8 "Skills alone: 8 skill files"
        assert_file_not_exists "$skill_dir/sdd-init/SKILL.md" "sdd-init NOT installed by skills alone"
        assert_file_exists "$skill_dir/go-testing/SKILL.md" "go-testing skill"
        assert_file_exists "$skill_dir/skill-creator/SKILL.md" "skill-creator skill"
        assert_file_not_exists "$skill_dir/issue-creation/SKILL.md" "issue-creation NOT installed by default"
        assert_file_size_min "$skill_dir/go-testing/SKILL.md" 200 "go-testing skill has real content"
    else
        log_fail "OpenCode skills (full) install command failed"
    fi
}

test_oc_context7_injection() {
    log_test "OpenCode: context7 injection (opencode.json MCP)"
    cleanup_test_env

    if $BINARY install --agent opencode --component context7 2>&1; then
        local settings="$HOME/.config/opencode/opencode.json"
        assert_file_exists "$settings" "OpenCode opencode.json"
        assert_file_contains "$settings" '"mcp"' "Has mcp key"
        assert_file_contains "$settings" '"context7"' "Has context7 entry"
        assert_file_contains "$settings" 'https://mcp.context7.com/mcp' "Has Context7 MCP URL"
        assert_valid_json "$settings" "opencode.json is valid JSON"
    else
        log_fail "OpenCode context7 install command failed"
    fi
}

# --- Category 4: Qwen Code injection ---

test_qwen_engram_injection() {
    log_test "Qwen: engram injection (settings.json)"
    cleanup_test_env

    if $BINARY install --agent qwen-code --component engram 2>&1; then
        local settings="$HOME/.qwen/settings.json"
        assert_file_exists "$settings" "Qwen settings.json"
        assert_file_contains "$settings" '"mcp"' "Has mcp key"
        assert_file_contains "$settings" '"engram"' "Has engram MCP entry"
        assert_file_contains "$settings" '"command"' "Has command key"
        assert_valid_json "$settings" "settings.json is valid JSON"
    else
        log_fail "Qwen engram install command failed"
    fi
}

test_qwen_engram_idempotency() {
    log_test "Qwen: engram injection is idempotent"
    cleanup_test_env

    local settings="$HOME/.qwen/settings.json"

    # First run — the install's exit code is irrelevant here (e.g. transient
    # npm failure); we assert on the resulting file.
    $BINARY install --agent qwen-code --component engram > /dev/null 2>&1 || true
    if [ ! -f "$settings" ]; then
        log_fail "Qwen settings.json missing after first install"
        return
    fi
    local checksum1
    checksum1=$(md5sum "$settings" | cut -d' ' -f1)

    # Second run
    $BINARY install --agent qwen-code --component engram > /dev/null 2>&1 || true
    if [ ! -f "$settings" ]; then
        log_fail "Qwen settings.json missing after second install"
        return
    fi
    local checksum2
    checksum2=$(md5sum "$settings" | cut -d' ' -f1)

    if [ "$checksum1" = "$checksum2" ]; then
        log_pass "Qwen settings.json is idempotent"
    else
        log_fail "Qwen settings.json changed between runs"
    fi
}

test_oc_permissions_injection() {
    log_test "OpenCode: permissions injection"
    cleanup_test_env

    if $BINARY install --agent opencode --component permissions 2>&1; then
        local settings="$HOME/.config/opencode/opencode.json"
        assert_file_exists "$settings" "OpenCode opencode.json"
        assert_file_contains "$settings" '"permission"' "Has permission key"
        assert_file_contains "$settings" '"bash"' "Has bash permissions"
        assert_file_contains "$settings" '"read"' "Has read permissions"
        assert_valid_json "$settings" "opencode.json is valid JSON"
    else
        log_fail "OpenCode permissions install command failed"
    fi
}

# --- Category 4: All components coexisting ---

test_all_components_claude_code() {
    log_test "Default component set: Claude Code (all components coexist)"
    cleanup_test_env

    # Engram needs binary install (go install) — may fail but injection components
    # that don't need binary install should be tested.
    # We test injection-only components first, then try the whole set.
    # If the whole set fails due to binary install, we fall back to individual injection-only test.
    if $BINARY install --agent claude-code --component sdd --component persona --component skills --component context7 --component permissions 2>&1; then
        local claude_md="$HOME/.claude/CLAUDE.md"
        local settings="$HOME/.claude/settings.json"

        # CLAUDE.md should have all 3 sections coexisting
        assert_file_exists "$claude_md" "CLAUDE.md exists"
        assert_file_contains_marker "$claude_md" "sdd-orchestrator" "Has SDD section"
        assert_file_contains_marker "$claude_md" "persona" "Has persona section"

        # No duplicate sections
        assert_no_duplicate_section "$claude_md" "sdd-orchestrator" "No duplicate SDD section"
        assert_no_duplicate_section "$claude_md" "persona" "No duplicate persona section"

        # settings.json should have permissions
        assert_file_exists "$settings" "settings.json exists"
        assert_file_contains "$settings" '"permissions"' "Has permissions"
        assert_valid_json "$settings" "settings.json is valid JSON"

        # MCP registration lives in the ~/.claude.json user registry, the only
        # user-scope file Claude Code reads MCP servers from (issue #1868).
        local registry="$HOME/.claude.json"
        assert_file_exists "$registry" "Claude user MCP registry (~/.claude.json)"
        assert_file_contains "$registry" '"mcpServers"' "Has MCP servers"
        assert_file_contains "$registry" '"context7"' "Has context7 MCP"
        assert_file_contains "$registry" 'context7-mcp' "Context7 MCP uses pinned package"
        assert_valid_json "$registry" "user registry is valid JSON"

        # Skills
        assert_file_count_min "$HOME/.claude/skills" "SKILL.md" 11 "At least 11 skill files"

        log_pass "Default set: all Claude Code injection-only components coexist"
    else
        log_fail "Default set (Claude Code) install command failed"
    fi
}

test_all_components_opencode() {
    log_test "Default component set: OpenCode (all components coexist)"
    cleanup_test_env

    if $BINARY install --agent opencode --component engram --component sdd --component persona --component skills --component context7 --component permissions 2>&1; then
        local settings="$HOME/.config/opencode/opencode.json"
        local agents_md="$HOME/.config/opencode/AGENTS.md"

        # opencode.json should have all overlays merged
        assert_file_exists "$settings" "OpenCode opencode.json"
        assert_file_contains "$settings" '"permission"' "Has permission config"
        assert_file_contains "$settings" '"mcp"' "Has MCP servers"
        assert_file_contains "$settings" '"context7"' "Has context7 MCP"
        assert_valid_json "$settings" "opencode.json is valid JSON"

        # AGENTS.md for persona + engram (SDD orchestrator is in opencode.json for OpenCode, NOT AGENTS.md)
        assert_file_exists "$agents_md" "AGENTS.md exists"
        assert_file_contains_marker "$agents_md" "engram-protocol" "AGENTS.md has engram protocol"
        assert_no_duplicate_section "$agents_md" "engram-protocol" "No duplicate engram section in AGENTS.md"
        # SDD orchestrator for OpenCode lives in opencode.json as an agent definition (not AGENTS.md)
        assert_file_matches "$settings" '"(axiom|gentle)-orchestrator"' "opencode.json has the managed orchestrator agent"
        assert_file_not_contains "$settings" '"sdd-orchestrator"' "opencode.json does not have legacy base sdd-orchestrator agent"
        # AGENTS.md must NOT have a sdd-orchestrator HTML section (it's handled by opencode.json)
        assert_file_not_contains_marker "$agents_md" "sdd-orchestrator" "AGENTS.md has no SDD section marker (opencode uses json agent)"

        # SDD commands
        assert_file_count_min "$HOME/.config/opencode/commands" "*.md" 7 "SDD command files"

        # Skills
        assert_file_count_min "$HOME/.config/opencode/skills" "SKILL.md" 11 "At least 11 skill files"

        log_pass "Default set: all OpenCode injection-only components coexist"
    else
        log_fail "Default set (OpenCode) install command failed"
    fi
}

test_ecosystem_both_agents() {
    log_test "Both agents with SDD, skills and context7"
    cleanup_test_env

    if $BINARY install --agent claude-code --agent opencode --component sdd --component skills --component context7 2>&1; then
        # Claude Code
        assert_file_exists "$HOME/.claude/CLAUDE.md" "Claude CLAUDE.md"
        assert_file_contains_marker "$HOME/.claude/CLAUDE.md" "sdd-orchestrator" "Claude has SDD"
        assert_file_contains "$HOME/.claude.json" '"context7"' "Claude context7 MCP"
        assert_file_count_min "$HOME/.claude/skills" "SKILL.md" 11 "Claude skills"

        # OpenCode
        assert_file_count_min "$HOME/.config/opencode/commands" "*.md" 7 "OpenCode SDD commands"
        assert_file_count_min "$HOME/.config/opencode/skills" "SKILL.md" 11 "OpenCode skills"
        assert_file_contains "$HOME/.config/opencode/opencode.json" '"context7"' "OpenCode context7"
        assert_valid_json "$HOME/.config/opencode/opencode.json" "OpenCode opencode.json valid JSON"

        log_pass "Both agents have matching components"
    else
        log_fail "Both agents install command failed"
    fi
}

test_both_agents_permissions() {
    log_test "Both agents: permissions injection"
    cleanup_test_env

    if $BINARY install --agent opencode --agent claude-code --component permissions 2>&1; then
        local oc_settings="$HOME/.config/opencode/opencode.json"
        local cc_settings="$HOME/.claude/settings.json"

        assert_file_exists "$oc_settings" "OpenCode opencode.json"
        assert_file_exists "$cc_settings" "Claude settings.json"
        assert_file_contains "$oc_settings" '"permission"' "OpenCode has permission config"
        assert_file_contains "$cc_settings" '"permissions"' "Claude has permissions"
        assert_valid_json "$oc_settings" "OpenCode opencode valid JSON"
        assert_valid_json "$cc_settings" "Claude settings valid JSON"
    else
        log_fail "Both agents + permissions install command failed"
    fi
}

# --- Category 5: Content validation ---

test_content_claude_md_sections_substantial() {
    log_test "Content validation: CLAUDE.md sections are substantial"
    cleanup_test_env

    # Install SDD + persona + engram (all inject into CLAUDE.md)
    $BINARY install --agent claude-code --component sdd --component persona 2>&1 || true
    $BINARY install --agent claude-code --component engram 2>&1 || true

    local claude_md="$HOME/.claude/CLAUDE.md"
    if [ -f "$claude_md" ]; then
        assert_file_size_min "$claude_md" 1000 "CLAUDE.md with 3 sections >= 1000 bytes"
    else
        log_fail "CLAUDE.md not created"
    fi
}

test_content_skills_are_real() {
    log_test "Content validation: skill files contain real instructions"
    cleanup_test_env

    $BINARY install --agent claude-code --component skills 2>&1 || true

    local skills_dir="$HOME/.claude/skills"
    if [ -d "$skills_dir" ]; then
        # Check every SKILL.md is at least 200 bytes (real content, not stubs)
        local all_ok=true
        while IFS= read -r skill_file; do
            local size
            size=$(wc -c < "$skill_file" | tr -d ' ')
            if [ "$size" -lt 200 ]; then
                log_fail "Skill file too small ($size bytes): $skill_file"
                all_ok=false
            fi
        done < <(find "$skills_dir" -name "SKILL.md" -type f)

        if $all_ok; then
            log_pass "All skill files have >= 200 bytes of real content"
        fi
    else
        log_fail "Skills directory not created"
    fi
}

test_content_mcp_json_valid() {
    log_test "Content validation: MCP JSON files are parseable"
    cleanup_test_env

    $BINARY install --agent claude-code --component context7 2>&1 || true
    $BINARY install --agent claude-code --component engram 2>&1 || true

    # Claude Code reads user-scoped MCP servers from ~/.claude.json. The legacy
    # ~/.claude/mcp directory is intentionally no longer created.
    local registry="$HOME/.claude.json"
    assert_file_exists "$registry" "Claude user MCP registry"
    assert_valid_json "$registry" "Claude user MCP registry is valid JSON"
    assert_file_contains "$registry" '"context7"' "Registry has Context7 server"
    assert_file_contains "$registry" '"engram"' "Registry has Engram server"
    assert_file_not_exists "$HOME/.claude/mcp/context7.json" "legacy Context7 MCP file is not written"
    assert_file_not_exists "$HOME/.claude/mcp/engram.json" "legacy Engram MCP file is not written"
}

test_content_opencode_commands_valid_markdown() {
    log_test "Content validation: OpenCode commands are valid markdown with frontmatter"
    cleanup_test_env

    $BINARY install --agent opencode --component sdd 2>&1 || true

    local commands_dir="$HOME/.config/opencode/commands"
    if [ -d "$commands_dir" ]; then
        local all_ok=true
        while IFS= read -r cmd_file; do
            local size
            size=$(wc -c < "$cmd_file" | tr -d ' ')
            if [ "$size" -lt 10 ]; then
                log_fail "Command file too small ($size bytes): $cmd_file"
                all_ok=false
            fi
        done < <(find "$commands_dir" -name "*.md" -type f)

        if $all_ok; then
            log_pass "All OpenCode command files have content"
        fi
    else
        log_fail "OpenCode commands directory not created"
    fi
}

# --- Category 6: Idempotency ---

test_idempotent_permissions_opencode() {
    log_test "Idempotency: permissions on OpenCode (run twice, same result)"
    cleanup_test_env

    $BINARY install --agent opencode --component permissions 2>&1 || true
    local first_hash
    first_hash=$(md5sum "$HOME/.config/opencode/opencode.json" 2>/dev/null | cut -d' ' -f1)

    $BINARY install --agent opencode --component permissions 2>&1 || true
    local second_hash
    second_hash=$(md5sum "$HOME/.config/opencode/opencode.json" 2>/dev/null | cut -d' ' -f1)

    if [ "$first_hash" = "$second_hash" ] && [ -n "$first_hash" ]; then
        log_pass "Idempotent: same permissions config after two runs"
    else
        log_fail "Permissions config changed between runs ($first_hash vs $second_hash)"
    fi
}

test_idempotent_sdd_claude() {
    log_test "Idempotency: SDD on Claude Code (no duplicate sections)"
    cleanup_test_env

    $BINARY install --agent claude-code --component sdd 2>&1 || true
    $BINARY install --agent claude-code --component sdd 2>&1 || true

    local claude_md="$HOME/.claude/CLAUDE.md"
    if [ -f "$claude_md" ]; then
        assert_no_duplicate_section "$claude_md" "sdd-orchestrator" "No duplicate SDD section after 2 runs"
    else
        log_fail "CLAUDE.md not found"
    fi
}

test_idempotent_persona_claude() {
    log_test "Idempotency: persona on Claude Code (no duplicate sections)"
    cleanup_test_env

    $BINARY install --agent claude-code --component persona 2>&1 || true
    $BINARY install --agent claude-code --component persona 2>&1 || true

    local claude_md="$HOME/.claude/CLAUDE.md"
    if [ -f "$claude_md" ]; then
        assert_no_duplicate_section "$claude_md" "persona" "No duplicate persona section after 2 runs"
    else
        log_fail "CLAUDE.md not found"
    fi
}

test_idempotent_engram_claude() {
    log_test "Idempotency: engram on Claude Code (no duplicate sections)"
    cleanup_test_env

    $BINARY install --agent claude-code --component engram 2>&1 || true
    $BINARY install --agent claude-code --component engram 2>&1 || true

    local claude_md="$HOME/.claude/CLAUDE.md"
    if [ -f "$claude_md" ]; then
        assert_no_duplicate_section "$claude_md" "engram-protocol" "No duplicate engram section after 2 runs"

        # Also check the user registry remains valid.
        local mcp_file="$HOME/.claude.json"
        if [ -f "$mcp_file" ]; then
            assert_valid_json "$mcp_file" "Claude user MCP registry still valid after 2 runs"
        fi
    else
        log_fail "CLAUDE.md not found"
    fi
}

# ─── Gemini parity tests ─────────────────────────────────────────────────────

test_gemini_engram_tools_flag() {
    log_test "Gemini: engram injection uses --tools=agent"
    cleanup_test_env

    if $BINARY install --agent gemini-cli --component engram 2>&1; then
        local settings="$HOME/.gemini/settings.json"
        assert_file_exists "$settings" "Gemini settings.json"
        assert_file_contains "$settings" '"mcpServers"' "Has mcpServers key"
        assert_file_contains "$settings" '"engram"' "Has engram entry"
        assert_file_contains "$settings" '"--tools=agent"' "Engram args include --tools=agent"
        assert_valid_json "$settings" "settings.json is valid JSON"
    else
        log_fail "Gemini engram install command failed"
    fi
}

# ─── Codex parity tests ───────────────────────────────────────────────────────

test_codex_engram_injection() {
    log_test "Codex: engram injection writes config.toml + instruction files"
    cleanup_test_env

    if $BINARY install --agent codex --component engram 2>&1; then
        local config_toml="$HOME/.codex/config.toml"
        local instructions="$HOME/.codex/engram-instructions.md"
        local compact="$HOME/.codex/engram-compact-prompt.md"

        assert_file_exists "$config_toml" "Codex config.toml"
        assert_file_contains "$config_toml" '[mcp_servers.engram]' "config.toml has [mcp_servers.engram]"
        assert_file_contains "$config_toml" 'command = ".*engram"' "config.toml has correct command"
        assert_file_contains "$config_toml" '"--tools=agent"' "config.toml has --tools=agent"
        assert_file_contains "$config_toml" 'model_instructions_file' "config.toml references instruction file"
        assert_file_contains "$config_toml" 'experimental_compact_prompt_file' "config.toml references compact prompt"

        assert_file_exists "$instructions" "engram-instructions.md"
        assert_file_contains "$instructions" 'mem_save' "Instructions have memory protocol content"

        assert_file_exists "$compact" "engram-compact-prompt.md"
        assert_file_contains "$compact" 'FIRST ACTION REQUIRED' "Compact prompt has required sentinel"
    else
        log_fail "Codex engram install command failed"
    fi
}

test_codex_engram_idempotent() {
    log_test "Codex: engram injection is idempotent (no duplicate blocks)"
    cleanup_test_env

    $BINARY install --agent codex --component engram 2>&1 || true
    $BINARY install --agent codex --component engram 2>&1 || true

    local config_toml="$HOME/.codex/config.toml"
    if [ -f "$config_toml" ]; then
        local count
        count=$(grep -c '\[mcp_servers\.engram\]' "$config_toml" || true)
        if [ "$count" -ne 1 ]; then
            log_fail "config.toml has $count [mcp_servers.engram] blocks after 2 runs (want exactly 1)"
        else
            log_pass "config.toml has exactly 1 [mcp_servers.engram] block after 2 runs"
        fi
    else
        log_fail "config.toml not found after 2 runs"
    fi
}

test_idempotent_skills_claude() {
    log_test "Idempotency: skills injection produces same files"
    cleanup_test_env

    $BINARY install --agent claude-code --component skills 2>&1 || true
    # Capture file hashes
    local first_hashes
    first_hashes=$(find "$HOME/.claude/skills" -name "SKILL.md" -exec md5sum {} \; 2>/dev/null | sort)

    $BINARY install --agent claude-code --component skills 2>&1 || true
    local second_hashes
    second_hashes=$(find "$HOME/.claude/skills" -name "SKILL.md" -exec md5sum {} \; 2>/dev/null | sort)

    if [ "$first_hashes" = "$second_hashes" ] && [ -n "$first_hashes" ]; then
        log_pass "Idempotent: same skill files after two runs"
    else
        log_fail "Skill files changed between runs"
    fi
}

test_idempotent_full_claude() {
    log_test "Idempotency: full injection-only on Claude Code"
    cleanup_test_env

    $BINARY install --agent claude-code --component sdd --component persona --component context7 --component permissions 2>&1 || true
    local first_md_hash
    first_md_hash=$(md5sum "$HOME/.claude/CLAUDE.md" 2>/dev/null | cut -d' ' -f1)
    # Snapshot settings.json for semantic comparison (engram setup may reorder
    # top-level keys on re-run — see engram binary's non-deterministic map
    # serialization). Byte-exact hashing would false-fail on harmless reorder.
    cp "$HOME/.claude/settings.json" /tmp/gai_settings_run1.json 2>/dev/null || true

    $BINARY install --agent claude-code --component sdd --component persona --component context7 --component permissions 2>&1 || true
    local second_md_hash
    second_md_hash=$(md5sum "$HOME/.claude/CLAUDE.md" 2>/dev/null | cut -d' ' -f1)

    if [ "$first_md_hash" = "$second_md_hash" ] && [ -n "$first_md_hash" ]; then
        log_pass "Idempotent: CLAUDE.md identical after 2 runs"
    else
        log_fail "CLAUDE.md changed between runs"
    fi
    if [ -f /tmp/gai_settings_run1.json ] && [ -f "$HOME/.claude/settings.json" ]; then
        if json_files_equal /tmp/gai_settings_run1.json "$HOME/.claude/settings.json"; then
            log_pass "Idempotent: settings.json identical after 2 runs"
        else
            log_fail "settings.json changed between runs"
        fi
    else
        log_fail "settings.json missing after install"
    fi
    rm -f /tmp/gai_settings_run1.json
}

# --- Category 8: Edge cases ---

test_edge_multiple_agents_same_component() {
    log_test "Edge case: multiple agents with same component"
    cleanup_test_env

    if $BINARY install --agent claude-code --agent opencode --component context7 2>&1; then
        # Both agents should have context7
        assert_file_contains "$HOME/.claude.json" '"context7"' "Claude context7"
        assert_file_contains "$HOME/.config/opencode/opencode.json" '"context7"' "OpenCode context7"
    else
        log_fail "Multiple agents + context7 install command failed"
    fi
}

test_edge_json_merge_preserves_existing() {
    log_test "Edge case: JSON merge preserves existing settings"
    cleanup_test_env

    # Create pre-existing settings
    mkdir -p "$HOME/.config/opencode"
    echo '{"existingKey": "preserved"}' > "$HOME/.config/opencode/opencode.json"

    # Install permissions on top
    $BINARY install --agent opencode --component permissions 2>&1 || true

    local settings="$HOME/.config/opencode/opencode.json"
    assert_file_contains "$settings" '"existingKey"' "Pre-existing key preserved"
    assert_file_contains "$settings" '"preserved"' "Pre-existing value preserved"
    assert_file_contains "$settings" '"permission"' "Permission config merged in"
    assert_valid_json "$settings" "Merged JSON is valid"
}

test_edge_multiple_json_overlays() {
    log_test "Edge case: multiple JSON overlays merge correctly"
    cleanup_test_env

    # Install permissions, then context7 — all into OpenCode opencode.json
    $BINARY install --agent opencode --component permissions 2>&1 || true
    $BINARY install --agent opencode --component context7 2>&1 || true

    local settings="$HOME/.config/opencode/opencode.json"
    assert_file_contains "$settings" '"permission"' "Permission config present after 2 merges"
    assert_file_contains "$settings" '"mcp"' "MCP servers present after 2 merges"
    assert_file_contains "$settings" '"context7"' "Context7 present after 2 merges"
    assert_valid_json "$settings" "Final merged JSON is valid"
}

# --- Category 10: Cursor agent files ---

test_cursor_sdd_subagents() {
    log_test "Cursor: SDD install writes 11 agent files to ~/.cursor/agents/"
    cleanup_test_env

    # Cursor is a desktop app — create the config dir to signal it's "installed"
    mkdir -p "$HOME/.cursor"

    if $BINARY install --agent cursor --component sdd 2>&1; then
        local agents_dir="$HOME/.cursor/agents"

        # Directory must exist
        assert_dir_exists "$agents_dir" "~/.cursor/agents/ directory"

        # All 11 SDD agent files must exist
        assert_file_exists "$agents_dir/sdd-init.md" "sdd-init.md agent file"
        assert_file_exists "$agents_dir/sdd-explore.md" "sdd-explore.md agent file"
        assert_file_exists "$agents_dir/sdd-research.md" "sdd-research.md agent file"
        assert_file_exists "$agents_dir/sdd-propose.md" "sdd-propose.md agent file"
        assert_file_exists "$agents_dir/sdd-spec.md" "sdd-spec.md agent file"
        assert_file_exists "$agents_dir/sdd-design.md" "sdd-design.md agent file"
        assert_file_exists "$agents_dir/sdd-tasks.md" "sdd-tasks.md agent file"
        assert_file_exists "$agents_dir/sdd-apply.md" "sdd-apply.md agent file"
        assert_file_exists "$agents_dir/sdd-verify.md" "sdd-verify.md agent file"
        assert_file_exists "$agents_dir/sdd-archive.md" "sdd-archive.md agent file"
        assert_file_exists "$agents_dir/sdd-onboard.md" "sdd-onboard.md agent file"

        # readonly flags: explore and verify are readonly: false (issue #156 — readonly: true
        # blocks MCP tools and terminal in Cursor, not just file writes)
        assert_file_contains "$agents_dir/sdd-explore.md" "readonly: false" "sdd-explore is not readonly"
        assert_file_contains "$agents_dir/sdd-verify.md" "readonly: false" "sdd-verify is not readonly"

        # apply must NOT be readonly (it writes code)
        assert_file_not_contains "$agents_dir/sdd-apply.md" "readonly: true" "sdd-apply is NOT readonly"

        # All agent files must have substantial content
        for phase in sdd-init sdd-explore sdd-research sdd-propose sdd-spec sdd-design sdd-tasks sdd-apply sdd-verify sdd-archive sdd-onboard; do
            assert_file_size_min "$agents_dir/$phase.md" 200 "$phase agent has real content"
        done
    else
        log_fail "Cursor SDD install command failed"
    fi
}

test_antigravity_sdd_skills_path() {
    log_test "Antigravity: SDD skills install to ~/.gemini/antigravity-cli/skills/"
    cleanup_test_env

    # Antigravity is a desktop app — create the config dir to signal it's "installed"
    mkdir -p "$HOME/.gemini/antigravity"

    if $BINARY install --agent antigravity --component sdd 2>&1; then
        local skills_dir="$HOME/.gemini/antigravity-cli/skills"
        assert_dir_exists "$skills_dir" "Antigravity skills directory"
        assert_file_exists "$skills_dir/sdd-init/SKILL.md" "sdd-init skill"
        assert_file_exists "$skills_dir/sdd-apply/SKILL.md" "sdd-apply skill"
        assert_file_exists "$skills_dir/_shared/sdd-phase-common.md" "shared convention"
        assert_file_size_min "$skills_dir/sdd-init/SKILL.md" 100 "skill has real content"

        # Path regression guard: skills must NOT go to legacy Gemini paths.
        if [ -d "$HOME/.gemini/skills/sdd-init" ]; then
            log_fail "Skills went to ~/.gemini/skills/ instead of ~/.gemini/antigravity-cli/skills/"
        elif [ -d "$HOME/.gemini/antigravity/skills/sdd-init" ]; then
            log_fail "Skills went to legacy ~/.gemini/antigravity/skills/ instead of ~/.gemini/antigravity-cli/skills/"
        else
            log_pass "Skills correctly in ~/.gemini/antigravity-cli/skills/"
        fi
    else
        log_fail "Antigravity SDD install command failed"
    fi
}

# --- Category 12: Codex context7 TOML injection ---

test_codex_context7_in_toml() {
    log_test "Codex: context7 component writes [mcp_servers.context7] into config.toml (TOML strategy)"
    cleanup_test_env

    $BINARY install --agent codex --component context7 2>&1 || true

    local config_toml="$HOME/.codex/config.toml"
    assert_file_exists "$config_toml" "Codex config.toml created by context7"
    assert_file_contains "$config_toml" "[mcp_servers.context7]" "Codex config.toml has [mcp_servers.context7] block"
    assert_file_contains "$config_toml" "https://mcp.context7.com/mcp" "Codex context7 block uses remote MCP URL"
    assert_file_not_contains "$config_toml" "context7-mcp" "Codex context7 block does not use local npx package"

    # Idempotent: re-running must not duplicate the block.
    $BINARY install --agent codex --component context7 2>&1 || true
    local count
    # `grep -c` prints "0" AND exits 1 on zero matches, so `|| echo 0` would
    # yield the two-line string "0\n0" and break the numeric comparison below.
    count=$(grep -c "\[mcp_servers.context7\]" "$config_toml" 2>/dev/null || true)
    count=${count:-0}
    if [ "$count" -eq 1 ]; then
        log_pass "Codex context7 block is idempotent (exactly 1 entry)"
    else
        log_fail "Codex context7 block duplicated ($count entries)"
    fi
}

# --- Category 7: Injection integrity (guards against issue #4 regression) ---

test_integrity_sdd_skills_nonempty() {
    log_test "Integrity: every SDD SKILL.md has real content (>100 bytes)"
    cleanup_test_env

    if $BINARY install --agent opencode --component sdd 2>&1; then
        local skill_dir="$HOME/.config/opencode/skills"
        local all_ok=true
        local sdd_skills=(sdd-init sdd-explore sdd-research sdd-propose sdd-spec sdd-design sdd-tasks sdd-apply sdd-verify sdd-archive sdd-onboard)

        for skill in "${sdd_skills[@]}"; do
            local path="$skill_dir/$skill/SKILL.md"
            if [ ! -f "$path" ]; then
                log_fail "SDD skill missing: $path"
                all_ok=false
                continue
            fi
            local size
            size=$(wc -c < "$path" | tr -d ' ')
            if [ "$size" -lt 100 ]; then
                log_fail "SDD skill empty or too small ($size bytes): $skill"
                all_ok=false
            fi
        done

        if $all_ok; then
            log_pass "All 11 SDD skills have >= 100 bytes of real content"
        fi
    else
        log_fail "SDD install command failed"
    fi
}

test_integrity_sdd_orchestrator_in_opencode_json() {
    log_test "Integrity: opencode.json contains gentle-orchestrator agent after SDD install"
    cleanup_test_env

    if $BINARY install --agent opencode --component sdd 2>&1; then
        local settings="$HOME/.config/opencode/opencode.json"
        assert_file_exists "$settings" "opencode.json exists"
        assert_file_matches "$settings" '"(axiom|gentle)-orchestrator"' "Has the managed orchestrator agent"
        assert_file_not_contains "$settings" '"sdd-orchestrator"' "Does not have legacy base sdd-orchestrator agent"
        assert_file_contains "$settings" '"agent"' "Has agent key"
        assert_valid_json "$settings" "opencode.json is valid JSON"
    else
        log_fail "SDD install for orchestrator check failed"
    fi
}

test_integrity_all_sdd_commands_have_frontmatter() {
    log_test "Integrity: all 11 SDD command files have YAML frontmatter"
    cleanup_test_env

    if $BINARY install --agent opencode --component sdd 2>&1; then
        local commands_dir="$HOME/.config/opencode/commands"
        local all_ok=true
        local expected_commands=(sdd-init sdd-apply sdd-archive sdd-continue sdd-explore sdd-ff sdd-new sdd-onboard sdd-research sdd-status sdd-verify)

        for cmd in "${expected_commands[@]}"; do
            local path="$commands_dir/$cmd.md"
            if [ ! -f "$path" ]; then
                log_fail "SDD command missing: $cmd.md"
                all_ok=false
                continue
            fi
            # Must start with --- (YAML frontmatter)
            if ! head -1 "$path" | grep -q '^---'; then
                log_fail "SDD command $cmd.md missing YAML frontmatter"
                all_ok=false
            fi
            # Must contain agent: sdd-orchestrator (except sdd-continue, sdd-ff, sdd-new which use different agent)
            local size
            size=$(wc -c < "$path" | tr -d ' ')
            if [ "$size" -lt 50 ]; then
                log_fail "SDD command $cmd.md too small ($size bytes)"
                all_ok=false
            fi
        done

        if $all_ok; then
            log_pass "All 11 SDD commands present with frontmatter and content"
        fi
    else
        log_fail "SDD install for command check failed"
    fi
}

test_integrity_all_skills_nonempty() {
    log_test "Integrity: every SKILL.md is non-empty"
    cleanup_test_env

    if $BINARY install --agent opencode --component sdd --component skills 2>&1; then
        local skill_dir="$HOME/.config/opencode/skills"
        local all_ok=true
        local empty_count=0

        while IFS= read -r skill_file; do
            local size
            size=$(wc -c < "$skill_file" | tr -d ' ')
            if [ "$size" -lt 100 ]; then
                log_fail "Skill file empty/corrupt ($size bytes): $skill_file"
                all_ok=false
                empty_count=$((empty_count + 1))
            fi
        done < <(find "$skill_dir" -name "SKILL.md" -type f)

        if $all_ok; then
            local total
            total=$(find "$skill_dir" -name "SKILL.md" -type f | wc -l | tr -d ' ')
            log_pass "All $total skill files have >= 100 bytes of real content"
        else
            log_fail "$empty_count skill file(s) are empty or corrupt"
        fi
    else
        log_fail "Skills install for integrity check failed"
    fi
}

test_integrity_sdd_orchestrator_agent_structure() {
    log_test "Integrity: gentle-orchestrator agent has required fields in opencode.json"
    cleanup_test_env

    if $BINARY install --agent opencode --component sdd 2>&1; then
        local settings="$HOME/.config/opencode/opencode.json"
        assert_file_matches "$settings" '"(axiom|gentle)-orchestrator"' "Has the managed orchestrator"
        assert_file_not_contains "$settings" '"sdd-orchestrator"' "Does not have legacy base sdd-orchestrator"
        assert_file_contains "$settings" '"mode"' "Agent has mode field"
        assert_file_contains "$settings" '"prompt"' "Agent has prompt field"
        assert_file_contains "$settings" 'COORDINATOR' "Agent prompt contains orchestrator instructions"
    else
        log_fail "SDD + persona install for agent structure check failed"
    fi
}

test_integrity_skills_plus_sdd_coexist() {
    log_test "Integrity: SDD + skills components write non-empty files that coexist"
    cleanup_test_env

    if $BINARY install --agent opencode --component sdd --component skills 2>&1; then
        local skill_dir="$HOME/.config/opencode/skills"

        # SDD skills should exist
        assert_file_size_min "$skill_dir/sdd-init/SKILL.md" 100 "sdd-init skill has content"
        assert_file_size_min "$skill_dir/sdd-apply/SKILL.md" 100 "sdd-apply skill has content"

        # Foundation skills should also exist
        assert_file_size_min "$skill_dir/go-testing/SKILL.md" 100 "go-testing skill has content"
        assert_file_size_min "$skill_dir/skill-creator/SKILL.md" 100 "skill-creator skill has content"

        # Shared SDD conventions should exist
        assert_file_exists "$skill_dir/_shared/persistence-contract.md" "Shared persistence contract"
        assert_file_size_min "$skill_dir/_shared/persistence-contract.md" 50 "Persistence contract has content"

        # opencode.json should have gentle-orchestrator as the base coordinator
        assert_file_matches "$HOME/.config/opencode/opencode.json" '"(axiom|gentle)-orchestrator"' "managed orchestrator present"
        assert_file_not_contains "$HOME/.config/opencode/opencode.json" '"sdd-orchestrator"' "legacy base sdd-orchestrator absent"
    else
        log_fail "SDD + skills coexistence install failed"
    fi
}

# --- Category 9: SDD multi-mode tests ---

test_oc_sdd_multi_mode_injection() {
    log_test "OpenCode: SDD multi-mode injection (10 agents in opencode.json)"
    cleanup_test_env

    if $BINARY install --agent opencode --component sdd --sdd-mode multi 2>&1; then
        local settings="$HOME/.config/opencode/opencode.json"
        local legacy_plugin="$HOME/.config/opencode/plugins/background-agents.ts"
        local model_variants_plugin="$HOME/.config/opencode/plugins/model-variants.ts"
        assert_file_exists "$settings" "opencode.json exists"
        assert_valid_json "$settings" "opencode.json is valid JSON"
        assert_file_matches "$settings" '"(axiom|gentle)-orchestrator"' "Has the managed orchestrator agent"
        assert_file_not_contains "$settings" '"sdd-orchestrator"' "Does not have legacy base sdd-orchestrator agent"
        assert_file_contains "$settings" '"sdd-apply"' "Has sdd-apply sub-agent"
        assert_file_contains "$settings" '"sdd-init"' "Has sdd-init sub-agent"
        assert_file_contains "$settings" '"sdd-verify"' "Has sdd-verify sub-agent"
        assert_file_contains "$settings" '"sdd-explore"' "Has sdd-explore sub-agent"
        assert_file_contains "$settings" '"sdd-propose"' "Has sdd-propose sub-agent"
        assert_file_contains "$settings" '"sdd-spec"' "Has sdd-spec sub-agent"
        assert_file_contains "$settings" '"sdd-design"' "Has sdd-design sub-agent"
        assert_file_contains "$settings" '"sdd-tasks"' "Has sdd-tasks sub-agent"
        assert_file_contains "$settings" '"sdd-archive"' "Has sdd-archive sub-agent"
        assert_file_contains "$settings" '"subagent"' "Sub-agents have mode subagent"
        assert_file_contains "$settings" '"task"' "Has native task tool"
        assert_file_not_exists "$legacy_plugin" "legacy background-agents plugin not installed by default"
        assert_file_exists "$model_variants_plugin" "model-variants plugin exists"
        assert_file_contains "$model_variants_plugin" 'model-variants' "Model variants plugin has expected content marker"
    else
        log_fail "OpenCode SDD multi-mode install command failed"
    fi
}

test_oc_sdd_single_mode_no_models() {
    log_test "OpenCode: SDD single mode has all agents but no model overrides"
    cleanup_test_env

    if $BINARY install --agent opencode --component sdd --sdd-mode single 2>&1; then
        local settings="$HOME/.config/opencode/opencode.json"
        assert_file_exists "$settings" "opencode.json exists"
        assert_valid_json "$settings" "opencode.json is valid JSON"
        assert_file_matches "$settings" '"(axiom|gentle)-orchestrator"' "Has the managed orchestrator agent"
        assert_file_not_contains "$settings" '"sdd-orchestrator"' "Single mode: does not have legacy base sdd-orchestrator agent"
        assert_file_contains "$settings" '"sdd-apply"' "Single mode: has sdd-apply sub-agent"
        assert_file_not_contains "$settings" '"model"' "Single mode: no model overrides"
        assert_file_not_exists "$HOME/.config/opencode/plugins/background-agents.ts" "Single mode: legacy background-agents plugin not installed"
        assert_file_exists "$HOME/.config/opencode/plugins/model-variants.ts" "Single mode: model-variants plugin present"
    else
        log_fail "OpenCode SDD single-mode install command failed"
    fi
}

test_oc_sdd_default_mode_same_as_single() {
    log_test "OpenCode: SDD default (no --sdd-mode flag) matches single mode"
    cleanup_test_env

    if $BINARY install --agent opencode --component sdd 2>&1; then
        local settings="$HOME/.config/opencode/opencode.json"
        assert_file_exists "$settings" "opencode.json exists"
        assert_file_matches "$settings" '"(axiom|gentle)-orchestrator"' "Has the managed orchestrator"
        assert_file_not_contains "$settings" '"sdd-orchestrator"' "Default mode: does not have legacy base sdd-orchestrator"
        assert_file_contains "$settings" '"sdd-apply"' "Default mode: has sdd-apply sub-agent"
        assert_file_not_contains "$settings" '"model"' "Default mode: no model overrides"
        assert_file_not_exists "$HOME/.config/opencode/plugins/background-agents.ts" "Default mode: legacy background-agents plugin not installed"
        assert_file_exists "$HOME/.config/opencode/plugins/model-variants.ts" "Default mode: model-variants plugin present"
    else
        log_fail "OpenCode SDD default mode install command failed"
    fi
}

# ===========================================================================
# TIER 3 — Backup / restore tests (require RUN_BACKUP_TESTS=1)
# ===========================================================================

test_backup_created_on_install() {
    log_test "Backup snapshot created during install"
    cleanup_test_env
    setup_fake_configs

    if $BINARY install --agent opencode --component permissions 2>&1; then
        local backup_count
        backup_count=$(find_backup_snapshots | wc -l | tr -d ' ')
        if [ "$backup_count" -gt 0 ]; then
            log_pass "Backup directory created ($backup_count snapshots)"
        else
            log_fail "No backup directory found"
        fi
    else
        log_fail "Install with backup failed"
    fi
}

test_backup_contains_original_files() {
    log_test "Backup snapshot contains original config files"
    cleanup_test_env
    setup_fake_configs

    if $BINARY install --agent opencode --component permissions 2>&1; then
        local latest_backup
        latest_backup=$(find_backup_snapshots | tail -1)
        if [ -n "$latest_backup" ]; then
            local file_count
            file_count=$(find "$latest_backup" -type f 2>/dev/null | wc -l | tr -d ' ')
            if [ "$file_count" -gt 0 ]; then
                log_pass "Backup contains $file_count file(s)"
            else
                log_fail "Backup directory is empty"
            fi
        else
            log_fail "No backup snapshot directory found"
        fi
    else
        log_fail "Install for backup test failed"
    fi
}

test_backup_manifest_exists() {
    log_test "Backup manifest file exists"
    cleanup_test_env
    setup_fake_configs

    if $BINARY install --agent opencode --component permissions 2>&1; then
        local latest_backup
        latest_backup=$(find_backup_snapshots | tail -1)
        if [ -n "$latest_backup" ]; then
            if [ -f "$latest_backup/manifest.json" ]; then
                assert_valid_json "$latest_backup/manifest.json" "Backup manifest is valid JSON"
            else
                log_fail "manifest.json not found in backup: $latest_backup"
            fi
        else
            log_fail "No backup snapshot found"
        fi
    else
        log_fail "Install for manifest test failed"
    fi
}

test_backup_idempotent_install() {
    log_test "Idempotent: running install twice produces same result (with backup)"
    cleanup_test_env

    $BINARY install --agent opencode --component permissions 2>&1 || true
    local first_content
    first_content=$(cat "$HOME/.config/opencode/opencode.json" 2>/dev/null)

    $BINARY install --agent opencode --component permissions 2>&1 || true
    local second_content
    second_content=$(cat "$HOME/.config/opencode/opencode.json" 2>/dev/null)

    if [ "$first_content" = "$second_content" ] && [ -n "$first_content" ]; then
        log_pass "Idempotent: same config after two runs (with backup)"
    else
        log_fail "Config changed between runs (with backup)"
    fi
}

test_backup_multiple_snapshots() {
    log_test "Multiple installs create multiple backup snapshots"
    cleanup_test_env
    setup_fake_configs

    $BINARY install --agent opencode --component permissions 2>&1 || true
    sleep 0.1
    $BINARY install --agent opencode --component context7 2>&1 || true

    local backup_count
    backup_count=$(find_backup_snapshots | wc -l | tr -d ' ')
    if [ "$backup_count" -ge 2 ]; then
        log_pass "Multiple backup snapshots created ($backup_count)"
    else
        log_fail "Expected >= 2 backup snapshots, got $backup_count"
    fi
}

test_backup_claude_code_files() {
    log_test "Backup captures Claude Code files"
    cleanup_test_env
    setup_fake_configs

    if $BINARY install --agent claude-code --component permissions 2>&1; then
        local latest_backup
        latest_backup=$(find_backup_snapshots | tail -1)
        if [ -n "$latest_backup" ] && [ -f "$latest_backup/manifest.json" ]; then
            log_pass "Claude Code backup snapshot with manifest created"
        else
            log_fail "No proper backup found for Claude Code install"
        fi
    else
        log_fail "Install for Claude backup test failed"
    fi
}

# ===========================================================================
# Test execution
# ===========================================================================

log_info "=== Tier 1: Basic binary & dry-run tests ==="

# Category 1a: Binary basics
test_binary_exists
test_binary_runs
test_version_command

# Category 1b: Dry-run output format
test_dry_run_output_format
test_dry_run_platform_detection
test_dry_run_detects_linux

# Category 1c: Agent flags
test_dry_run_agent_claude_code
test_dry_run_agent_opencode
test_dry_run_agent_both
test_dry_run_agent_csv

# Category 1d: Default component set
test_dry_run_default_persona_before_sdd
test_dry_run_explicit_components

# Category 1f: Individual component flags (all 8)
test_dry_run_component_engram
test_dry_run_component_sdd
test_dry_run_component_skills
test_dry_run_component_context7
test_dry_run_component_persona
test_dry_run_component_permissions

# Category 1f2: SDD mode flag
test_dry_run_sdd_mode_multi
test_dry_run_sdd_mode_single
test_dry_run_sdd_mode_invalid_rejected

# Category 1g: Invalid inputs
test_invalid_component_rejected
test_unknown_command_rejected

if [ "${RUN_FULL_E2E:-0}" = "1" ]; then
    log_info ""
    log_info "=== Tier 2: Component injection tests ==="

    # Category 2: Claude Code injection
    test_cc_engram_injection
    test_cc_sdd_injection
    test_cc_skills_full
    test_cc_custom_skills_with_flag
    test_cc_custom_sdd_plus_skills
    test_cc_context7_injection
    test_cc_permissions_injection

    # Category 3: OpenCode injection
    test_oc_engram_injection
    test_oc_sdd_injection
    test_oc_skills_full
    test_oc_context7_injection
    test_oc_permissions_injection

    # Category 4: All components coexisting
    test_all_components_claude_code
    test_all_components_opencode
    test_ecosystem_both_agents
    test_both_agents_permissions

    # Category 5: Content validation
    test_content_claude_md_sections_substantial
    test_content_skills_are_real
    test_content_mcp_json_valid
    test_content_opencode_commands_valid_markdown

    # Category 6: Idempotency
    test_idempotent_permissions_opencode
    test_idempotent_sdd_claude
    test_idempotent_persona_claude
    test_idempotent_engram_claude
    test_idempotent_skills_claude
    test_idempotent_full_claude

    # Category 6b: Gemini/Codex engram parity
    test_gemini_engram_tools_flag
    test_codex_engram_injection
    test_codex_engram_idempotent

    # Category 8: Edge cases
    test_edge_multiple_agents_same_component
    test_edge_json_merge_preserves_existing
    test_edge_multiple_json_overlays

    # Category 7: Injection integrity (issue #4 regression guard)
    test_integrity_sdd_skills_nonempty
    test_integrity_sdd_orchestrator_in_opencode_json
    test_integrity_all_sdd_commands_have_frontmatter
    test_integrity_all_skills_nonempty
    test_integrity_sdd_orchestrator_agent_structure
    test_integrity_skills_plus_sdd_coexist

    # Category 9: SDD multi-mode
    test_oc_sdd_multi_mode_injection
    test_oc_sdd_single_mode_no_models
    test_oc_sdd_default_mode_same_as_single

    # Category 10: Cursor native agent files
    test_cursor_sdd_subagents

    # Antigravity skills path
    test_antigravity_sdd_skills_path

    # Category 12: Codex context7 by-design skip
    test_codex_context7_in_toml

    # Category 13: Qwen integration
    test_qwen_engram_injection
    test_qwen_engram_idempotency
else
    log_skip "Tier 2 tests (set RUN_FULL_E2E=1 to enable)"
fi

if [ "${RUN_BACKUP_TESTS:-0}" = "1" ]; then
    log_info ""
    log_info "=== Tier 3: Backup/restore tests ==="
    test_backup_created_on_install
    test_backup_contains_original_files
    test_backup_manifest_exists
    test_backup_idempotent_install
    test_backup_multiple_snapshots
    test_backup_claude_code_files
else
    log_skip "Tier 3 tests (set RUN_BACKUP_TESTS=1 to enable)"
fi

# ---------------------------------------------------------------------------
# Summary & exit
# ---------------------------------------------------------------------------
print_summary
