# Minecraft Server Management App

My first Go application. Provides automated management for a Minecraft server, including:

- Automatic server shutdown when idle (no players online for 15 minutes)
- Discord bot integration for server control and status
- Player count monitoring
- Extensible console output handlers

## Features

- Idle watcher: Saves resources by stopping the server when no players are online
- Discord bot: Start, stop, and query server status via Discord commands
- Nix build: Reproducible builds using Nix and Go modules
- Configurable working directory

## Usage

### Running the App

```sh
./minecraft-app-manager --workingPath /path/to/server/folder
```

- `--workingPath`: Path to use as the working directory (where your Minecraft server files are)

### Discord Bot

- Set the `DISCORD_BOT_TOKEN` environment variable before running
- Use Discord commands like `!start`, `!status`, `!players`, `!help` in your server's Discord channel

## Extending

- You can register custom console handlers in Go to listen for specific server output

## License

MIT
