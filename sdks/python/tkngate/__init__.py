from openai import OpenAI
import os

def wrap(client: OpenAI, virtual_key: str = None, proxy_url: str = "http://localhost:7477/openai/v1", provider: str = "openai", session_id: str = None) -> OpenAI:
    """
    Wraps an existing OpenAI client to route traffic through the Tkngate proxy.
    
    :param client: The instantiated openai.OpenAI client
    :param virtual_key: The Tkngate virtual key (e.g., 'tkngate-sk-...'). 
                        If None, it looks for the TKNGATE_VIRTUAL_KEY environment variable.
    :param proxy_url: The URL where the Tkngate proxy is running.
    :param provider: The upstream provider to use.
    :param session_id: The session ID for budgeting.
    :return: The modified OpenAI client
    """
    key = virtual_key or os.environ.get("TKNGATE_VIRTUAL_KEY")
    if not key:
        raise ValueError("A Tkngate virtual key must be provided either via the virtual_key parameter or the TKNGATE_VIRTUAL_KEY environment variable.")
    
    client.base_url = proxy_url
    client.api_key = key
    
    if client.default_headers is None:
        client.default_headers = {}
    
    if provider:
        client.default_headers['X-Tkngate-Provider'] = provider
    if session_id:
        client.default_headers['X-Tkngate-Session-ID'] = session_id
        
    return client

def wrap_anthropic(client, virtual_key: str = None, proxy_url: str = "http://localhost:7477/anthropic/v1", provider: str = "anthropic", session_id: str = None):
    """
    Wraps an existing Anthropic client to route traffic through the Tkngate proxy.
    """
    key = virtual_key or os.environ.get("TKNGATE_VIRTUAL_KEY")
    if not key:
        raise ValueError("A Tkngate virtual key must be provided either via the virtual_key parameter or the TKNGATE_VIRTUAL_KEY environment variable.")
    
    client.base_url = proxy_url
    client.api_key = key
    
    if client.default_headers is None:
        client.default_headers = {}
        
    if provider:
        client.default_headers['X-Tkngate-Provider'] = provider
    if session_id:
        client.default_headers['X-Tkngate-Session-ID'] = session_id
        
    return client
