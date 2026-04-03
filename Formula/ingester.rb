class Ingester < Formula
  desc "Semantic search file ingester for ChromaDB"
  homepage "https://github.com/yetanotherchris/ingester"
  version "VERSION"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/yetanotherchris/ingester/releases/download/vVERSION/ingester-darwin-arm64.tar.gz"
      sha256 "SHA256"
    else
      url "https://github.com/yetanotherchris/ingester/releases/download/vVERSION/ingester-darwin-amd64.tar.gz"
      sha256 "SHA256"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/yetanotherchris/ingester/releases/download/vVERSION/ingester-linux-arm64.tar.gz"
      sha256 "SHA256"
    else
      url "https://github.com/yetanotherchris/ingester/releases/download/vVERSION/ingester-linux-amd64.tar.gz"
      sha256 "SHA256"
    end
  end

  def install
    bin.install "ingester"
  end

  test do
    assert_match "ingester version", shell_output("#{bin}/ingester --version")
  end
end
