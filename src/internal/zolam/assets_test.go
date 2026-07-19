package zolam

import "testing"

func TestOnnxRuntimeReleaseAssetFor(t *testing.T) {
	cases := []struct {
		goos, arch string
		want       string
	}{
		{"linux", "amd64", "onnxruntime-linux-x64-" + onnxRuntimeVersion + ".tgz"},
		{"linux", "arm64", "onnxruntime-linux-aarch64-" + onnxRuntimeVersion + ".tgz"},
		{"darwin", "arm64", "onnxruntime-osx-arm64-" + onnxRuntimeVersion + ".tgz"},
		{"windows", "amd64", "onnxruntime-win-x64-" + onnxRuntimeVersion + ".zip"},
		{"windows", "arm64", "onnxruntime-win-arm64-" + onnxRuntimeVersion + ".zip"},
	}
	for _, tc := range cases {
		got, err := onnxRuntimeReleaseAssetFor(tc.goos, tc.arch)
		if err != nil {
			t.Errorf("onnxRuntimeReleaseAssetFor(%q, %q) returned error: %v", tc.goos, tc.arch, err)
			continue
		}
		if got != tc.want {
			t.Errorf("onnxRuntimeReleaseAssetFor(%q, %q) = %q, want %q", tc.goos, tc.arch, got, tc.want)
		}
	}
}

func TestOnnxRuntimeReleaseAssetFor_UnsupportedCombos(t *testing.T) {
	unsupported := []struct{ goos, arch string }{
		{"darwin", "amd64"}, // Intel Mac: no longer published upstream for this version
		{"linux", "386"},
		{"windows", "386"},
		{"freebsd", "amd64"},
	}
	for _, tc := range unsupported {
		if _, err := onnxRuntimeReleaseAssetFor(tc.goos, tc.arch); err == nil {
			t.Errorf("onnxRuntimeReleaseAssetFor(%q, %q) expected error, got nil", tc.goos, tc.arch)
		}
	}
}
