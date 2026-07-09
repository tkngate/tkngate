import { ClientOptions } from 'openai';

/**
 * Wraps an existing OpenAI client configuration to route traffic through the Tkngate proxy.
 *
 * @param options - The instantiated OpenAI client options
 * @param virtualKey - The Tkngate virtual key (e.g., 'tkngate-sk-...'). If omitted, looks for TKNGATE_VIRTUAL_KEY environment variable.
 * @param proxyUrl - The URL where the Tkngate proxy is running. Defaults to "http://localhost:7477/openai/v1"
 * @returns The modified OpenAI client options
 */
export function wrapConfig(
  options?: ClientOptions,
  virtualKey?: string | null,
  proxyUrl?: string
): ClientOptions;
