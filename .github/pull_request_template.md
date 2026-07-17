## User value

Describe the problem solved and the behavior users will observe.

## Changes

- _List the key implementation changes._

## Security and data impact

- [ ] No new network exposure, command construction, executable path, or file-system risk
- [ ] Persistence is backward-compatible or includes an explicit migration
- [ ] No credentials, cookies, profile data, or personal information are included
- [ ] New dependencies are justified and reflected in notices/SBOM inputs

Explain any checked item that needs context:

## Verification

- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `pnpm --dir frontend test`
- [ ] `pnpm --dir frontend build`
- [ ] `powershell -ExecutionPolicy Bypass -File scripts/check-lines.ps1`
- [ ] Manual launch/stop check performed when browser runtime behavior changed

## Documentation and presentation

- [ ] Tests cover the behavior change
- [ ] User-facing documentation and `CHANGELOG.md` are updated
- [ ] UI changes include redacted screenshots when useful
- [ ] Unsupported or partially applied settings remain visible as warnings
