package encryption

import (
	"fmt"
	"sync"
)

// SecurityHandlerFactoryInstance is the singleton instance of the factory.
//
// Port of SecurityHandlerFactory.INSTANCE.
var SecurityHandlerFactoryInstance = newSecurityHandlerFactory()

// SecurityHandlerFactory manufactures security handlers.
//
// Port of org.apache.pdfbox.pdmodel.encryption.SecurityHandlerFactory, which
// keeps a class per name and per policy class and builds one by reflection. Go
// has no such constructor lookup, so the registry keeps the two constructor
// functions instead; registerHandler takes them where Java takes the classes.
type SecurityHandlerFactory struct {
	mu              sync.Mutex
	nameToHandler   map[string]func() SecurityHandler
	policyToHandler map[string]func(policy ProtectionPolicy) SecurityHandler
}

func newSecurityHandlerFactory() *SecurityHandlerFactory {
	f := &SecurityHandlerFactory{
		nameToHandler:   map[string]func() SecurityHandler{},
		policyToHandler: map[string]func(ProtectionPolicy) SecurityHandler{},
	}
	f.mustRegisterHandler(StandardSecurityHandlerFilter,
		func() SecurityHandler { return NewStandardSecurityHandler() },
		(&StandardProtectionPolicy{}).policyKey(),
		func(policy ProtectionPolicy) SecurityHandler {
			return NewStandardSecurityHandlerOfPolicy(policy.(*StandardProtectionPolicy))
		})
	f.mustRegisterHandler(PublicKeySecurityHandlerFilter,
		func() SecurityHandler { return NewPublicKeySecurityHandler() },
		(&PublicKeyProtectionPolicy{}).policyKey(),
		func(policy ProtectionPolicy) SecurityHandler {
			return NewPublicKeySecurityHandlerOfPolicy(policy.(*PublicKeyProtectionPolicy))
		})
	return f
}

// RegisterHandler registers a security handler.
//
// If the given filter name is already registered, an error is returned; Java
// throws IllegalStateException. A policy that is already registered is *not*
// refused — it is silently replaced. Java's javadoc promises otherwise ("If
// another handler was previously registered for the same filter name or for the
// same policy name, an exception is thrown") but its code only looks in
// nameToHandler, so the port does the same. See migration/JAVA-BUGS.md 26.
func (f *SecurityHandlerFactory) RegisterHandler(name string,
	newForFilter func() SecurityHandler, policyKey string,
	newForPolicy func(ProtectionPolicy) SecurityHandler) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, present := f.nameToHandler[name]; present {
		return fmt.Errorf("The security handler name is already registered")
	}
	// JAVA BUG 26: policyToHandler is overwritten without a check, though the
	// javadoc above the Java method says a duplicate policy throws.
	f.nameToHandler[name] = newForFilter
	f.policyToHandler[policyKey] = newForPolicy
	return nil
}

func (f *SecurityHandlerFactory) mustRegisterHandler(name string,
	newForFilter func() SecurityHandler, policyKey string,
	newForPolicy func(ProtectionPolicy) SecurityHandler) {
	if err := f.RegisterHandler(name, newForFilter, policyKey, newForPolicy); err != nil {
		panic(err)
	}
}

// NewSecurityHandlerForPolicy returns a new security handler for the given
// protection policy, or nil where none is registered for it.
func (f *SecurityHandlerFactory) NewSecurityHandlerForPolicy(
	policy ProtectionPolicy) SecurityHandler {
	f.mu.Lock()
	newForPolicy, present := f.policyToHandler[policy.policyKey()]
	f.mu.Unlock()
	if !present {
		return nil
	}
	return newForPolicy(policy)
}

// NewSecurityHandlerForFilter returns a new security handler for the given
// /Filter name, or nil where none is registered under it.
func (f *SecurityHandlerFactory) NewSecurityHandlerForFilter(name string) SecurityHandler {
	f.mu.Lock()
	newForFilter, present := f.nameToHandler[name]
	f.mu.Unlock()
	if !present {
		return nil
	}
	return newForFilter()
}
