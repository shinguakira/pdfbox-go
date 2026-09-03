// Package pdfio provides the random-access reading and writing primitives that
// the rest of the library is built on.
//
// It is the Go port of the Java package org.apache.pdfbox.io (the "io" Maven
// module). The package is named pdfio rather than io so that ported code can
// use the standard library's io package without an import alias; see
// migration/conventions/java-to-go.md for the full naming rules.
package pdfio
