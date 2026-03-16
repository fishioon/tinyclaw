export type AgentRuntimeMode = 'echo' | 'claude_agent_sdk';

export interface AgentEnv {
  serverPort: number;
  anthropicApiKey?: string;
  anthropicBaseUrl?: string;
  claudeCodeOauthToken?: string;
  agentIdleAfterSec: number;
  agentLogLevel: string;
  claudeRuntimeTimeoutMs: number;
  agentWorkdir: string;
  agentTmpdir: string;
  agentRuntimeMode: AgentRuntimeMode;
  claudeModel: string;
  claudeSystemPromptAppend?: string;
  claudeAllowedTools?: string[];
  claudeDisallowedTools?: string[];
  claudeMaxTurns: number;
  executeTimeoutMs: number;
}

export interface AgentRequest {
  msgid: string;
  roomId: string;
  tenantId: string;
  chatType: string;
  query: string;
}

export interface ExecutionResult {
  stdout: string;
  stderr: string;
  exit_code: number;
}

export interface FileEntry {
  name: string;
  size: number;
  type: 'file' | 'directory';
  mod_time: number;
}
