/**
 * Wraps an existing OpenAI client configuration to route traffic through the Tkngate proxy.
 *
 * @param {import('openai').ClientOptions} options - The instantiated OpenAI client options
 * @param {string} [virtualKey] - The Tkngate virtual key (e.g., 'tkngate-sk-...').
 *                                If omitted, looks for TKNGATE_VIRTUAL_KEY environment variable.
 * @param {string} [proxyUrl="http://localhost:7477/openai/v1"] - The URL where the Tkngate proxy is running.
 * @returns {import('openai').ClientOptions} The modified OpenAI client options
 */
function wrapConfig(options = {}, virtualKey = null, proxyUrl = "http://localhost:7477/openai/v1") {
  const key = virtualKey || process.env.TKNGATE_VIRTUAL_KEY;
  if (!key) {
    throw new Error(
      "A Tkngate virtual key must be provided either via the virtualKey parameter or the TKNGATE_VIRTUAL_KEY environment variable."
    );
  }

  return {
    ...options,
    baseURL: proxyUrl,
    apiKey: key,
  };
}

module.exports = {
  wrapConfig,
};
