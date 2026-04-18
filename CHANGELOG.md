# Changelog

All notable changes to this project should be documented in this file.

## Unreleased

## v0.2.2 - 2026-04-18

- improved `metrics compile` output with structured LLM analyze results instead of escaped JSON strings
- added `--expr-description` and env support so expression descriptions can be passed into compile analysis
- made compiled metric and group summaries more readable while keeping stable event and evidence fields

- extracted the Clinic SDK into its own repository
- kept the package name `clinicapi`
- added typed request and response models
- added typed error classification helpers
- added request lifecycle hooks
- added public repository scaffolding
