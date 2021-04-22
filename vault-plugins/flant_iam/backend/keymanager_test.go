package backend

import "testing"

func Test_nested_keyManeger(t *testing.T) {
	parent := &keyManager{
		idField:   "pid",
		entryName: "parent",
	}

	child := &keyManager{
		idField:   "cid",
		entryName: "child",
		parent:    parent,
	}

	grandchild := &keyManager{
		idField:   "gid",
		entryName: "grandchild",
		parent:    child,
	}

	{
		want := "parent(/(?P<pid>\\w(([\\w-.]+)?\\w)?))?"
		got := parent.EntryPattern()
		if want != got {
			t.Fatalf("parent entry pattern: got=%v, want=%v", got, want)
		}
	}

	{
		want := "parent/?"
		got := parent.ListPattern()
		if want != got {
			t.Fatalf("parent list pattern: got=%v, want=%v", got, want)
		}
	}

	{
		want := "parent/(?P<pid>\\w(([\\w-.]+)?\\w)?)/child(/(?P<cid>\\w(([\\w-.]+)?\\w)?))?"
		got := child.EntryPattern()
		if want != got {
			t.Fatalf("child entry pattern: got=%v, want=%v", got, want)
		}
	}

	{
		want := "parent/(?P<pid>\\w(([\\w-.]+)?\\w)?)/child/?"
		got := child.ListPattern()
		if want != got {
			t.Fatalf("child list pattern: got=%v, want=%v", got, want)
		}
	}

	{
		want := "parent/(?P<pid>\\w(([\\w-.]+)?\\w)?)/child/(?P<cid>\\w(([\\w-.]+)?\\w)?)/grandchild(/(?P<gid>\\w(([\\w-.]+)?\\w)?))?"
		got := grandchild.EntryPattern()
		if want != got {
			t.Fatalf("grandchild entry pattern: got=%v, want=%v", got, want)
		}
	}

	{
		want := "parent/(?P<pid>\\w(([\\w-.]+)?\\w)?)/child/(?P<cid>\\w(([\\w-.]+)?\\w)?)/grandchild/?"
		got := grandchild.ListPattern()
		if want != got {
			t.Fatalf("grandchild list pattern: got=%v, want=%v", got, want)
		}
	}
}
