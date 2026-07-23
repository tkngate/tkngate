# frozen_string_literal: true

Gem::Specification.new do |spec|
  spec.name = "tkngate"
  spec.version = "1.0.0"
  spec.authors = ["Tkngate"]
  spec.email = ["hello@tkngate.com"]

  spec.summary = "Zero-trust P2P gateway for LLM agents"
  spec.description = "Official Ruby middleware SDK for Tkngate. This acts as a transparent interceptor that automatically routes your Faraday traffic through the sidecar and injects the necessary security credentials."
  spec.homepage = "https://github.com/tkngate/tkngate"
  spec.license = "Apache-2.0"
  spec.required_ruby_version = ">= 2.6.0"

  spec.metadata["homepage_uri"] = spec.homepage
  spec.metadata["source_code_uri"] = "https://github.com/tkngate/tkngate/tree/main/sdks/ruby"
  spec.metadata["changelog_uri"] = "https://github.com/tkngate/tkngate/blob/main/CHANGELOG.md"

  # Specify which files should be added to the gem when it is released.
  spec.files = Dir["lib/**/*"]
  spec.require_paths = ["lib"]

  spec.add_dependency "faraday", ">= 1.0", "< 3.0"
end
