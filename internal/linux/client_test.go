package linux

import "testing"

func TestSafePath(t *testing.T) {
	for _, x := range []struct {
		p  string
		ok bool
	}{{"/tmp/a", true}, {"relative", false}, {"/tmp/../etc", false}} {
		{
			if safePath(x.p) != x.ok {
				t.Fatalf("%s", x.p)
			}
		}
	}
}

func TestSafeService(t *testing.T) {
	for _, service := range []string{"nginx", "docker.service", "app@1"} {
		if !ValidServiceName(service) {
			t.Fatalf("rejected %q", service)
		}
	}
	if ValidServiceName("nginx; reboot") {
		t.Fatal("accepted shell input")
	}
}
