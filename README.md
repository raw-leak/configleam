# Configleam

Configleam is an open-source project aimed at providing a dynamic and efficient way to manage and synchronize configuration files for microservices. It is designed to work natively within Kubernetes environments, leveraging Git repositories for storing and retrieving configurations.

## Table of Contents

- [Status](#status)
- [Installation](#installation)
- [Usage](#usage)
  - [Configuration of dedicated git repository](#configuration-of-dedicated-git-repository)
- [Contributing](#contributing)
- [License](#license)
- [Contact](#contact)

## Status

This project is currently `in-progress``. I am actively developing its core features and invite contributions and feedback from the community.

## Installation

Instructions on how to install and set up configleam will be provided as the project matures. For now, you can clone the repository to keep track of the latest developments.

```bash
git clone https://github.com/yourusername/configleam.git
cd configleam
# Future installation steps will be added here.
```

## Usage

// TODO

### Configuration of dedicated git repository

Configleam is designed to manage service configurations efficiently, with a focus on grouping related settings for easy access and management. It distinguishes between global configurations, which are broadly applicable, and local configurations, which are specific to and contained within named groups.

#### Configuration Types

1. **Global:**
    - Broadly applicable settings across different contexts.
    - Can be defined in any file without a specific naming convention.
    - Examples: include general service settings, default database configurations, application-wide feature flags, etc.

2. **Groups:** 
    - Named collections (groups) of configurations that aggregate global settings and group-specific local settings.
    - Groups serve as a single point of access for all related configurations, both global and local to that group.
    - Group names are prefixed with `group:` to signify their special role in collecting configurations.
   

3. **Local:**
    - Context-specific settings contained within a group.
    - Local configurations only exist within the context of their respective groups and are used to override global settings or add new group-specific settings.

#### Example Configurations
##### Global Configurations

```yaml
# Example of global configurations

database:
  type: sql
  host: global-db-host
  port: 3306

featureFlags:
  betaFeatures: false
  darkMode: true
```

In this example:
`database` and `featureFlags` are global configurations. They define the default database settings and application-wide feature flags.

#### Group Configurations
```yaml
# Example of group configurations with both global and local variables

group:analytics:
  - featureFlags
  - database: 
      host: analytics-db-host
      port: 3307
  - additionalMetrics: true

group:marketing:
  - database
  - featureFlags:
      betaFeatures: true
  - marketingCampaignsEnabled: true

```
In this example: 

Analytics Group (`group:analytics`): Inherits the global featureFlags and modifies the global database settings for its specific needs. It also includes an analytics-specific setting additionalMetrics.

Marketing Group (`group:marketing:`): Inherits the global database configuration and overrides the featureFlags setting. It introduces a marketing-specific setting marketingCampaignsEnabled.

### Notes

- Global configurations act as default settings. They apply broadly unless overridden by a group-specific configuration.
- Local configurations allow for flexibility and customization within specific groups or contexts.
- Configleam processes these configurations to apply the appropriate settings based on their global or group-specific nature.

## Contributing

Your contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any contributions you make to configleam are **greatly appreciated**.

1. Fork the project
2. Create your feature branch (`git checkout -b feature/YourFeature`)
3. Commit your changes (`git commit -m 'Add some YourFeature'`)
4. Push to the branch (`git push origin feature/YourFeature`)
5. Open a pull request

## License

Distributed under the MIT License. See `LICENSE` for more information.

## Contact

[LinkedIn](https://www.linkedin.com/in/mykhaylo-gusak/)

[Mykhaylo Gusak] - mykhaylogusak@hotmail.com

Project Link: [https://github.com/raw-leak/configleam](https://github.com/raw-leak/configleam)
