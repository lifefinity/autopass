# This file goes in a separate repo: lifefinity/homebrew-tap
# Users install via: brew install lifefinity/tap/autopass

class Autopass < Formula
  desc "Encrypted expect in one binary - auto-answer interactive prompts"
  homepage "https://github.com/lifefinity/autopass"
  version "0.2.0"  # Update on release
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/lifefinity/autopass/releases/download/v#{version}/autopass_#{version}_darwin_arm64.tar.gz"
      # sha256 "UPDATE_ON_RELEASE"
    end
    on_intel do
      url "https://github.com/lifefinity/autopass/releases/download/v#{version}/autopass_#{version}_darwin_amd64.tar.gz"
      # sha256 "UPDATE_ON_RELEASE"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/lifefinity/autopass/releases/download/v#{version}/autopass_#{version}_linux_arm64.tar.gz"
      # sha256 "UPDATE_ON_RELEASE"
    end
    on_intel do
      url "https://github.com/lifefinity/autopass/releases/download/v#{version}/autopass_#{version}_linux_amd64.tar.gz"
      # sha256 "UPDATE_ON_RELEASE"
    end
  end

  def install
    bin.install "autopass"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/autopass version")
  end
end
