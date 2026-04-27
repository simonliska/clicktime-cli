### clicktime-cli
Unofficial interactive prompt for time tracking in ClickUp

### Motivation 
Main motivation for me was to save some time

### Install
- Install Go: https://go.dev/doc/install
- Install `clicktime-cli`:
```sh
go install github.com/simonliska/clicktime-cli@latest
```
- Run `clicktime-cli --version` to verify installation
- Run `clicktime-cli` to start tracking time
- On first run you will be prompted for: 
- `ClickUP API key`: ClickUp Setting -> ClickUP API
- `team-id` aka `workspace-id`: From ClickUp URL: https://app.clickup.com/'your_workspace_id'
- `email`: for filtering assigned tasks

### Demo
![](./docs/demo.gif)