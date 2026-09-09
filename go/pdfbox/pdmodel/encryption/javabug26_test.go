package encryption

// JAVA-BUGS 26: `SecurityHandlerFactory.registerHandler` promises in its
// javadoc to refuse a duplicate filter name *or* a duplicate policy, and
// checks only the name. In-package, because the factory constructor and the
// policy key are.

import "testing"

// TestRegisterHandlerRefusesADuplicatePolicy is the defect.
//
// The expected behaviour is the javadoc's: "If another handler was previously
// registered for the same filter name or for the same policy name, an
// exception is thrown". The standard policy is registered by the constructor,
// so a second registration naming it must be refused whatever filter name it
// arrives under, and must leave the policy pointing where it did.
func TestRegisterHandlerRefusesADuplicatePolicy(t *testing.T) {
	// the registration mutates the factory, so this builds its own rather than
	// reaching for the singleton the other tests use
	factory := newSecurityHandlerFactory()
	standardPolicy := NewStandardProtectionPolicy("o", "u", NewAccessPermission())

	err := factory.RegisterHandler("Other.Filter",
		func() SecurityHandler { return NewPublicKeySecurityHandler() },
		standardPolicy.policyKey(),
		func(ProtectionPolicy) SecurityHandler { return NewPublicKeySecurityHandler() })
	if err == nil {
		t.Error("a policy that is already registered was accepted a second time")
	}

	// the policy keeps the handler it had
	forPolicy := factory.NewSecurityHandlerForPolicy(standardPolicy)
	if _, ok := forPolicy.(*StandardSecurityHandler); !ok {
		t.Errorf("the standard policy maps to %T, want the handler it was registered with",
			forPolicy)
	}
	// and the refused registration left no half-registered name behind
	if got := factory.NewSecurityHandlerForFilter("Other.Filter"); got != nil {
		t.Errorf("the refused filter name resolves to %T, want nothing", got)
	}
}

// TestRegisterHandlerStillRefusesADuplicateName keeps the check the Java does
// make, so the fix adds one refusal rather than moving it.
func TestRegisterHandlerStillRefusesADuplicateName(t *testing.T) {
	factory := newSecurityHandlerFactory()
	err := factory.RegisterHandler(StandardSecurityHandlerFilter,
		func() SecurityHandler { return NewPublicKeySecurityHandler() },
		"a.policy.key.nothing.else.uses",
		func(ProtectionPolicy) SecurityHandler { return NewPublicKeySecurityHandler() })
	if err == nil {
		t.Error("a filter name that is already registered was accepted a second time")
	}
}
