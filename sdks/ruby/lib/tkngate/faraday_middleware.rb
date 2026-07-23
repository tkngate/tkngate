require 'faraday'

module Tkngate
  class FaradayMiddleware < Faraday::Middleware
    def initialize(app, options = {})
      super(app)
      @virtual_key = options[:virtual_key] || ENV['TKNGATE_VIRTUAL_KEY']
      @provider = options[:provider]
      @session_id = options[:session_id]
    end

    def call(env)
      if @virtual_key && !@virtual_key.empty?
        env[:request_headers]['Authorization'] = "Bearer #{@virtual_key}"
      end

      if @provider && !@provider.empty?
        env[:request_headers]['X-Tkngate-Provider'] = @provider
      end

      if @session_id && !@session_id.empty?
        env[:request_headers]['X-Tkngate-Session-ID'] = @session_id
      end

      @app.call(env)
    end
  end
end

Faraday::Request.register_middleware tkngate: -> { Tkngate::FaradayMiddleware }
