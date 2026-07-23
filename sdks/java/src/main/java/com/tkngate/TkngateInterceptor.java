package com.tkngate;

import okhttp3.Interceptor;
import okhttp3.Request;
import okhttp3.Response;
import java.io.IOException;

public class TkngateInterceptor implements Interceptor {
    private final String virtualKey;
    private final String provider;
    private final String sessionId;

    public TkngateInterceptor(String virtualKey, String provider, String sessionId) {
        this.virtualKey = virtualKey != null ? virtualKey : System.getenv("TKNGATE_VIRTUAL_KEY");
        this.provider = provider;
        this.sessionId = sessionId;
    }

    @Override
    public Response intercept(Chain chain) throws IOException {
        Request original = chain.request();
        Request.Builder requestBuilder = original.newBuilder();

        if (this.virtualKey != null && !this.virtualKey.isEmpty()) {
            requestBuilder.header("Authorization", "Bearer " + this.virtualKey);
        }

        if (this.provider != null && !this.provider.isEmpty()) {
            requestBuilder.header("X-Tkngate-Provider", this.provider);
        }

        if (this.sessionId != null && !this.sessionId.isEmpty()) {
            requestBuilder.header("X-Tkngate-Session-ID", this.sessionId);
        }

        Request request = requestBuilder.build();
        return chain.proceed(request);
    }
}
