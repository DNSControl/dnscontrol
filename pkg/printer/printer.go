package printer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// CLI is an abstraction around the CLI.
type CLI interface {
	Printer
	StartDomain(dc *models.DomainConfig)
	StartDNSProvider(name string, skip bool)
	EndProvider(name string, numCorrections int, err error)
	EndProvider2(name string, numCorrections int)
	StartRegistrar(name string, skip bool)

	PrintCorrection(n int, c *models.Correction)
	PrintReport(n int, c *models.Correction) // Print corrections that are diff2.REPORT
	EndCorrection(err error)
	PromptToRun() bool
}

// Printer is a simple abstraction for printing data. Can be passed to providers to give simple output capabilities.
type Printer interface {
	Debugf(fmt string, args ...any)
	Printf(fmt string, args ...any)
	Println(lines ...string)
	Warnf(fmt string, args ...any)
	Errorf(fmt string, args ...any)
	PrintfIf(prnt bool, fmt string, args ...any)
}

// Debugf is called to print/format debug information.
func Debugf(fmt string, args ...any) {
	DefaultPrinter.Debugf(fmt, args...)
}

// Printf is called to print/format information.
func Printf(fmt string, args ...any) {
	DefaultPrinter.Printf(fmt, args...)
}

// Println is called to print/format information.
func Println(lines ...string) {
	DefaultPrinter.Println(lines...)
}

// Warnf is called to print/format a warning.
func Warnf(fmt string, args ...any) {
	DefaultPrinter.Warnf(fmt, args...)
}

// Errorf is called to print/format an error.
// func Errorf(fmt string, args ...any) {
// 	DefaultPrinter.Errorf(fmt, args...)
// }

// PrintfIf is called to optionally print something.
func PrintfIf(prnt bool, fmt string, args ...any) {
	DefaultPrinter.PrintfIf(prnt, fmt, args...)
}

// DefaultPrinter is the default Printer, used by Debugf, Printf, and Warnf.
var DefaultPrinter = &ConsolePrinter{
	Reader:  bufio.NewReader(os.Stdin),
	Writer:  os.Stdout,
	Verbose: false,
}

// SkinnyReport is true to disable certain print statements.
// This is a hack until we have the new printer replacement. The long
// variable name is easy to grep for when we make the conversion.
var SkinnyReport = true

// MaxReport represents how many records to show if SkinnyReport == true.
var MaxReport = 5

// ConsolePrinter is a handle for the console printer.
type ConsolePrinter struct {
	Reader *bufio.Reader
	Writer io.Writer

	Verbose bool
}

// StartDomain is called at the start of each domain.
func (c ConsolePrinter) StartDomain(dc *models.DomainConfig) {
	fmt.Fprintf(c.Writer, "******************** Domain: %s\n", dc.DisplayName)
}

// PrintCorrection is called to print/format each correction.
func (c ConsolePrinter) PrintCorrection(i int, correction *models.Correction) {
	fmt.Fprintf(c.Writer, "#%d: %s\n", i+1, correction.Msg)
}

// PrintReport is called to print/format each non-mutating correction (diff2.REPORT).
func (c ConsolePrinter) PrintReport(i int, correction *models.Correction) {
	fmt.Fprintf(c.Writer, "INFO#%d: %s\n", i+1, correction.Msg)
}

// PromptToRun prompts the user to see if they want to execute a correction.
func (c ConsolePrinter) PromptToRun() bool {
	fmt.Fprint(c.Writer, "Run? (y/N): ")
	txt, err := c.Reader.ReadString('\n')
	run := err == nil
	txt = strings.ToLower(strings.TrimSpace(txt))
	if txt != "y" {
		run = false
	}
	if !run {
		fmt.Fprintln(c.Writer, "Skipping")
	}
	return run
}

// EndCorrection is called at the end of each correction.
func (c ConsolePrinter) EndCorrection(err error) {
	if err != nil {
		fmt.Fprintln(c.Writer, "FAILURE!", err)
	} else {
		fmt.Fprintln(c.Writer, "SUCCESS!")
	}
}

// StartDNSProvider is called at the start of each new provider.
func (c ConsolePrinter) StartDNSProvider(provider string, skip bool) {
	lbl := ""
	if skip {
		lbl = " (skipping)"
	}
	if !SkinnyReport {
		fmt.Fprintf(c.Writer, "----- DNS Provider: %s...%s\n", provider, lbl)
	}
}

// StartRegistrar is called at the start of each new registrar.
func (c ConsolePrinter) StartRegistrar(provider string, skip bool) {
	lbl := ""
	if skip {
		lbl = " (skipping)"
	}
	if !SkinnyReport {
		fmt.Fprintf(c.Writer, "----- Registrar: %s...%s\n", provider, lbl)
	}
}

// EndProvider is called at the end of each provider.
func (c ConsolePrinter) EndProvider(name string, numCorrections int, err error) {
	if err != nil {
		fmt.Fprintln(c.Writer, "ERROR")
		fmt.Fprintf(c.Writer, "Error getting corrections (%s): %s\n", name, err)
	} else {
		plural := "s"
		if numCorrections == 1 {
			plural = ""
		}
		if (SkinnyReport) && (numCorrections == 0) {
			return
		}
		fmt.Fprintf(c.Writer, "%d correction%s (%s)\n", numCorrections, plural, name)
	}
}

// EndProvider2 is called at the end of each provider.
func (c ConsolePrinter) EndProvider2(name string, numCorrections int) {
	plural := "s"
	if numCorrections == 1 {
		plural = ""
	}
	if (SkinnyReport) && (numCorrections == 0) {
		return
	}
	fmt.Fprintf(c.Writer, "%d correction%s (%s)\n", numCorrections, plural, name)
}

// Debugf is called to print/format debug information.
func (c ConsolePrinter) Debugf(format string, args ...any) {
	if c.Verbose {
		fmt.Fprintf(c.Writer, format, args...)
	}
}

// Printf is called to print/format information.
func (c ConsolePrinter) Printf(format string, args ...any) {
	fmt.Fprintf(c.Writer, format, args...)
}

// Println is called to print/format information.
func (c ConsolePrinter) Println(lines ...string) {
	fmt.Fprintln(c.Writer, lines)
}

// Warnf is called to print/format a warning.
func (c ConsolePrinter) Warnf(format string, args ...any) {
	fmt.Fprintf(c.Writer, "WARNING: "+format, args...)
}

// Errorf is called to print/format an error.
func (c ConsolePrinter) Errorf(format string, args ...any) {
	fmt.Fprintf(c.Writer, "ERROR: "+format, args...)
}

// PrintfIf is called to optionally print/format a message.
func (c ConsolePrinter) PrintfIf(prnt bool, format string, args ...any) {
	if prnt {
		fmt.Fprintf(c.Writer, format, args...)
	}
}
