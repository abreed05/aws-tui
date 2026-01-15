# aws-tui

Terminal UI for browsing AWS resources. Built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Supported Resources

EC2, VPC, Security Groups, IAM (Users/Roles/Policies), RDS, ECS, Lambda, S3, KMS, Secrets Manager

## Requirements

- Go 1.21+
- AWS credentials configured (`~/.aws/credentials` or environment variables)

## Build

```bash
make build
```

Binary outputs to `./bin/aws-tui`.

## Run

```bash
./bin/aws-tui
```

Or with a specific profile:
```bash
AWS_PROFILE=myprofile ./bin/aws-tui
```

## Usage

| Key | Action |
|-----|--------|
| `:` | Command mode |
| `p` | Switch profile |
| `R` | Switch region |
| `j/k` | Navigate |
| `enter` | Select |
| `d` | Describe resource |
| `/` | Search |
| `esc` | Back |
| `q` | Quit |

Commands: `:users`, `:roles`, `:policies`, `:ec2`, `:vpc`, `:sg`, `:rds`, `:ecs`, `:lambda`, `:s3`, `:kms`, `:secrets`

## Themes

Config file: `~/.config/aws-tui/config.yaml`

```yaml
theme: default  # options: default, dark, light, nord, dracula
```

### Custom Themes

Create `~/.config/aws-tui/themes/<name>.yaml`:

```yaml
name: mytheme
colors:
  primary: "39"
  secondary: "62"
  accent: "212"
  background: "235"
  foreground: "252"
  muted: "245"
  success: "78"
  warning: "214"
  error: "196"
  info: "81"
  border: "240"
  selection: "57"
  selection_fg: "229"
```

Then set `theme: mytheme` in config.yaml. Colors use 256-color ANSI codes.
