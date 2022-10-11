
# Portbrella Whitelist Provider

Use this Traefik plugin to declares IP Whitelist middlewares that synchronize to your Portbrella IP lists.

## Usage

For a plugin to be active for a given Traefik instance, it must be declared in the static configuration.

Plugins are parsed and loaded exclusively during startup, which allows Traefik to check the integrity of the code and catch errors early on.
If an error occurs during loading, the plugin is disabled.

For security reasons, it is not possible to start a new plugin or modify an existing one while Traefik is running.

### Configuration

The Traefik static configuration must define the module name.

The following declaration (given here in YAML) defines the plugin:

```yaml
# Static configuration

experimental:
  plugins:
    portbrella:
      moduleName: github.com/portbrella/traefik_whitelist
      version: v1.0.2

providers:
  plugin:
    portbrella:
      mylistname: your_list_id_here
```

