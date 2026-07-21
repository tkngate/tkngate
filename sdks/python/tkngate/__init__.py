from openai import OpenAI
import os

def wrap(client: OpenAI, virtual_key: str = None, proxy_url: str = "http://localhost:7477/openai/v1") -> OpenAI:
    """
    Wraps an existing OpenAI client to route traffic through the Tkngate proxy.
    
    :param client: The instantiated openai.OpenAI client
    :param virtual_key: The Tkngate virtual key (e.g., 'tkngate-sk-...'). 
                        If None, it looks for the TKNGATE_VIRTUAL_KEY environment variable.
    :param proxy_url: The URL where the Tkngate proxy is running.
    :return: The modified OpenAI client
    """
    key = virtual_key or os.environ.get("TKNGATE_VIRTUAL_KEY")
    if not key:
        raise ValueError("A Tkngate virtual key must be provided either via the virtual_key parameter or the TKNGATE_VIRTUAL_KEY environment variable.")
    
    # Override the base URL to route traffic locally
    client.base_url = proxy_url
    
    # Override the API key with the virtual key
    client.api_key = key
    
    return client
