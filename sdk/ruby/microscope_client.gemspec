Gem::Specification.new do |spec|
  spec.name          = "microscope_client"
  spec.version       = "0.1.0"
  spec.summary       = "Thin HTTP client for the microscope observability API"
  spec.description   = "Record and query microscope observability entries over HTTP."
  spec.authors       = ["Qobly"]
  spec.license       = "MIT"
  spec.homepage      = "https://github.com/qobly/microscope"
  spec.required_ruby_version = ">= 3.0"
  spec.files         = Dir["lib/**/*.rb"]
  spec.require_paths = ["lib"]
end
