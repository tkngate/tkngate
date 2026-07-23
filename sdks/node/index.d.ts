import { ClientOptions } from 'openai';

export interface TkngateOptions {
  provider?: string;
  sessionId?: string;
}

/**
 * Wraps an existing OpenAI client configuration to route traffic through the Tkngate proxy.
 *
 * @param options - The instantiated OpenAI client options
 * @param virtualKey - The Tkngate virtual key (e.g., 'tkngate-sk-...'). If omitted, looks for TKNGATE_VIRTUAL_KEY environment variable.
 * @param proxyUrl - The URL where the Tkngate proxy is running.
 * @param tkngateOptions - Additional Tkngate-specific options (e.g. provider, sessionId).
 * @returns The modified OpenAI client options
 */
export function wrapConfig(
  options?: any,
  virtualKey?: string | null,
  proxyUrl?: string,
  tkngateOptions?: TkngateOptions
): any;

/**
 * Wraps an existing Anthropic client configuration to route traffic through the Tkngate proxy.
 *
 * @param options - The instantiated Anthropic client options
 * @param virtualKey - The Tkngate virtual key (e.g., 'tkngate-sk-...'). If omitted, looks for TKNGATE_VIRTUAL_KEY environment variable.
 * @param proxyUrl - The URL where the Tkngate proxy is running.
 * @param tkngateOptions - Additional Tkngate-specific options (e.g. provider, sessionId).
 * @returns The modified Anthropic client options
 */
export function wrapAnthropicConfig(
  options?: any,
  virtualKey?: string | null,
  proxyUrl?: string,
  tkngateOptions?: TkngateOptions
): any;
