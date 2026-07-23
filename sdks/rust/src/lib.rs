use reqwest::{header::{HeaderMap, HeaderValue, AUTHORIZATION}, Client, Request, Response};
use reqwest_middleware::{Middleware, Next, Result as MiddlewareResult};
use std::env;

pub struct TkngateMiddleware {
    virtual_key: Option<String>,
    provider: Option<String>,
    session_id: Option<String>,
}

impl TkngateMiddleware {
    pub fn new(virtual_key: Option<String>, provider: Option<String>, session_id: Option<String>) -> Self {
        let key = virtual_key.or_else(|| env::var("TKNGATE_VIRTUAL_KEY").ok());
        Self {
            virtual_key: key,
            provider,
            session_id,
        }
    }
}

#[async_trait::async_trait]
impl Middleware for TkngateMiddleware {
    async fn handle(
        &self,
        mut req: Request,
        extensions: &mut task_local_extensions::Extensions,
        next: Next<'_>,
    ) -> MiddlewareResult<Response> {
        let headers = req.headers_mut();
        
        if let Some(key) = &self.virtual_key {
            if let Ok(value) = HeaderValue::from_str(&format!("Bearer {}", key)) {
                headers.insert(AUTHORIZATION, value);
            }
        }
        
        if let Some(prov) = &self.provider {
            if let Ok(value) = HeaderValue::from_str(prov) {
                headers.insert("X-Tkngate-Provider", value);
            }
        }
        
        if let Some(sess) = &self.session_id {
            if let Ok(value) = HeaderValue::from_str(sess) {
                headers.insert("X-Tkngate-Session-ID", value);
            }
        }

        next.run(req, extensions).await
    }
}
