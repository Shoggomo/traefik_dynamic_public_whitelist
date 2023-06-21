
# Public Dynamic IP Whitelist Plugin

Use this Traefik plugin to create a dynamic IP Whitelist middleware that synchronizes to your public IP.

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
    traefik_dynamic_public_whitelist:
      moduleName: github.com/Shoggomo/traefik_dynamic_public_whitelist
      version: v0.1.1
```

You must restart Traefik.

# Dynamic configuration

In your dynamic configuration, let's say with a Docker label, you can use that middleware:

```
labels:
  - traefik.http.routers.my-router.middlewares=dpw_middleware@plugin-traefik_dynamic_public_whitelist
```

