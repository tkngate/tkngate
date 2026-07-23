<?php

namespace Tkngate;

use Psr\Http\Message\RequestInterface;

class TkngateMiddleware
{
    private $virtualKey;
    private $provider;
    private $sessionId;

    public function __construct($virtualKey = null, $provider = null, $sessionId = null)
    {
        $this->virtualKey = $virtualKey ?: getenv('TKNGATE_VIRTUAL_KEY');
        $this->provider = $provider;
        $this->sessionId = $sessionId;
    }

    public function __invoke(callable $handler)
    {
        return function (RequestInterface $request, array $options) use ($handler) {
            if (!empty($this->virtualKey)) {
                $request = $request->withHeader('Authorization', 'Bearer ' . $this->virtualKey);
            }

            if (!empty($this->provider)) {
                $request = $request->withHeader('X-Tkngate-Provider', $this->provider);
            }

            if (!empty($this->sessionId)) {
                $request = $request->withHeader('X-Tkngate-Session-ID', $this->sessionId);
            }

            return $handler($request, $options);
        };
    }
}
