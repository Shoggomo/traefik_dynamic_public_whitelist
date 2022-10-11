
# Portbrella IP Whitelist Provider

Use this Traefik plugin to create IP Whitelist middleware that synchronizes to your Portbrella IP lists.

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
      list1: your_list_id_here
      list2: another_list_id_here
```


At that point, Traefik has created list1 and list2 IP Whitelist middleware that can be used to filter traffic.

# Dynamic configuration

In your dynamic configuration, let say with Docker label, you can use that middleware:

```
labels:
  - "traefik.http.routers.your_service.middlewares=list1@plugin-portbrella"
```

In the previous code, replace "your_service" by your service name and replace "list1" by any list you declared in static configuration.