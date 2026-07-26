require "minitest/autorun"
require "minitest/mock"
require_relative "../lib/microscope_client"

class ClientTest < Minitest::Test
  def setup
    @client = MicroscopeClient::Client.new(base_url: "http://localhost:8093/microscope/")
  end

  def test_record_posts_name_and_content
    response = Minitest::Mock.new
    response.expect(:is_a?, true, [Net::HTTPSuccess])
    response.expect(:body, JSON.generate({ id: "entry-1" }))

    http = Minitest::Mock.new
    http.expect(:use_ssl=, nil, [false])
    http.expect(:read_timeout=, nil, [5])
    http.expect(:request, response, [Net::HTTP::Post])

    Net::HTTP.stub(:new, http) do
      id = @client.record("payment_charged", content: { amount: 4200 })
      assert_equal "entry-1", id
    end

    http.verify
  end

  def test_raises_on_non_success_response
    response = Minitest::Mock.new
    response.expect(:is_a?, false, [Net::HTTPSuccess])
    response.expect(:code, "500")

    http = Minitest::Mock.new
    http.expect(:use_ssl=, nil, [false])
    http.expect(:read_timeout=, nil, [5])
    http.expect(:request, response, [Net::HTTP::Post])

    Net::HTTP.stub(:new, http) do
      assert_raises(MicroscopeClient::Error) do
        @client.record("boom")
      end
    end
  end
end
