class Gotssh < Formula
  desc "功能强大的SSH连接和端口转发管理工具"
  homepage "https://github.com/MinatoHikari/gotssh"
  version "1.1.0"
  
  if Hardware::CPU.arm?
    url "https://github.com/MinatoHikari/gotssh/releases/download/v1.1.0/gotssh-1.1.0-darwin-arm64.tar.gz"
    sha256 "1e6264ef50619bdc291118812e577f39fb8a44131dac32ed534e639d5649fb9e"
  else
    url "https://github.com/MinatoHikari/gotssh/releases/download/v1.1.0/gotssh-1.1.0-darwin-amd64.tar.gz"
    sha256 "880d7e5b80a7c0c9647f07a93458dc34a0ae84c17911df72fa7e8387e959fb46"
  end
  
  depends_on "go" => :build
  
  def install
    bin.install "gotssh-darwin-arm64" => "gotssh" if Hardware::CPU.arm?
    bin.install "gotssh-darwin-amd64" => "gotssh" if Hardware::CPU.intel?
  end
  
  test do
    system "#{bin}/gotssh", "-h"
  end
end
