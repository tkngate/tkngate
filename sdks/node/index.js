/**
 * Wraps an existing OpenAI client configuration to route traffic through the Tkngate proxy.
 *
 * @param {import('openai').ClientOptions} options - The instantiated OpenAI client options
 * @param {string} [virtualKey] - The Tkngate virtual key (e.g., 'tkngate-sk-...').
 *                                If omitted, looks for TKNGATE_VIRTUAL_KEY environment variable.
 * @param {string} [proxyUrl="http://localhost:7477/openai/v1"] - The URL where the Tkngate proxy is running.
 * @param {Object} [tkngateOptions={}] - Additional Tkngate options
 * @param {string} [tkngateOptions.provider="openai"] - The upstream provider to use
 * @param {string} [tkngateOptions.sessionId] - The session ID for budgeting
 * @returns {import('openai').ClientOptions} The modified OpenAI client options
 */
function wrapConfig(options = {}, virtualKey = null, proxyUrl = "http://localhost:7477/openai/v1", tkngateOptions = {}) {
  const key = virtualKey || process.env.TKNGATE_VIRTUAL_KEY;
  if (!key) {
    throw new Error(
      "A Tkngate virtual key must be provided either via the virtualKey parameter or the TKNGATE_VIRTUAL_KEY environment variable."
    );
  }

  const defaultHeaders = { ...(options.defaultHeaders || {}) };
  if (tkngateOptions.provider) {
    defaultHeaders['X-Tkngate-Provider'] = tkngateOptions.provider;
  }
  if (tkngateOptions.sessionId) {
    defaultHeaders['X-Tkngate-Session-ID'] = tkngateOptions.sessionId;
  }

  return {
    ...options,
    baseURL: proxyUrl,
    apiKey: key,
    defaultHeaders,
  };
}

/**
 * Wraps an existing Anthropic client configuration to route traffic through the Tkngate proxy.
 *
 * @param {import('@anthropic-ai/sdk').ClientOptions} options - The instantiated Anthropic client options
 * @param {string} [virtualKey] - The Tkngate virtual key (e.g., 'tkngate-sk-...').
 *                                If omitted, looks for TKNGATE_VIRTUAL_KEY environment variable.
 * @param {string} [proxyUrl="http://localhost:7477/anthropic/v1"] - The URL where the Tkngate proxy is running.
 * @param {Object} [tkngateOptions={}] - Additional Tkngate options
 * @param {string} [tkngateOptions.provider="anthropic"] - The upstream provider to use
 * @param {string} [tkngateOptions.sessionId] - The session ID for budgeting
 * @returns {import('@anthropic-ai/sdk').ClientOptions} The modified Anthropic client options
 */
function wrapAnthropicConfig(options = {}, virtualKey = null, proxyUrl = "http://localhost:7477/anthropic/v1", tkngateOptions = { provider: 'anthropic' }) {
  const key = virtualKey || process.env.TKNGATE_VIRTUAL_KEY;
  if (!key) {
    throw new Error(
      "A Tkngate virtual key must be provided either via the virtualKey parameter or the TKNGATE_VIRTUAL_KEY environment variable."
    );
  }

  const defaultHeaders = { ...(options.defaultHeaders || {}) };
  if (tkngateOptions.provider) {
    defaultHeaders['X-Tkngate-Provider'] = tkngateOptions.provider;
  }
  if (tkngateOptions.sessionId) {
    defaultHeaders['X-Tkngate-Session-ID'] = tkngateOptions.sessionId;
  }

  return {
    ...options,
    baseURL: proxyUrl,
    apiKey: key, // Anthropic uses apiKey under the hood
    defaultHeaders,
  };
}

module.exports = {
  wrapConfig,
  wrapAnthropicConfig,
};
