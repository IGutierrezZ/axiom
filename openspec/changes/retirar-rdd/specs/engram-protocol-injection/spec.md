# Delta for engram-protocol-injection

## ADDED Requirements

### Requirement: Engram Protocol Handles Ambiguous Project (REQ-46.10)

The Engram MCP integration protocol and guidelines MUST document and provide for the orderly handling of the `ambiguous_project` state in the initial handshake.
(Moved from REQ-18.6 of `rdd-decoupling-v3-stability`, retired with RDD (C-01-e).)

#### Scenario: Project ambiguity in the Engram session handshake

- **GIVEN** a multi-project environment with Engram MCP
- **WHEN** the agent detects an ambiguous-project indication
- **THEN** the protocol instructs the agent to specify the project identifier unambiguously with `--project=<name>` before retrying
