using System;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;

namespace Tkngate
{
    public class TkngateHandler : DelegatingHandler
    {
        private readonly string _virtualKey;
        private readonly string _provider;
        private readonly string _sessionId;

        public TkngateHandler(string virtualKey = null, string provider = null, string sessionId = null)
        {
            _virtualKey = virtualKey ?? Environment.GetEnvironmentVariable("TKNGATE_VIRTUAL_KEY");
            _provider = provider;
            _sessionId = sessionId;
        }

        protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
        {
            if (!string.IsNullOrEmpty(_virtualKey))
            {
                request.Headers.Authorization = new System.Net.Http.Headers.AuthenticationHeaderValue("Bearer", _virtualKey);
            }

            if (!string.IsNullOrEmpty(_provider))
            {
                request.Headers.Add("X-Tkngate-Provider", _provider);
            }

            if (!string.IsNullOrEmpty(_sessionId))
            {
                request.Headers.Add("X-Tkngate-Session-ID", _sessionId);
            }

            return await base.SendAsync(request, cancellationToken);
        }
    }
}
