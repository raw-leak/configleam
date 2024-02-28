# Configleam 

Configleam is an open-source project aimed at providing a dynamic, simple, and efficient way to manage and synchronize configuration files for microservices. It is designed to work natively within Kubernetes environments, leveraging Git repositories for storing and retrieving configurations.

## Table of Contents

- [Status](#status)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration Repository Structure](#configuration-repository-structure)
- [Security](#security)
- [Contributing](#contributing)
- [Kubernetes Integration](#kubernetes-integration)
- [License](#license)
- [Contact](#contact)

## Status

This project is currently `in-progress`. I am actively developing its core features and invite contributions and feedback from the community.

## Installation

Instructions on how to install and set up configleam will be provided as the project matures. For now, you can clone the repository to keep track of the latest developments.

```bash
git clone https://github.com/raw-leak/configleam.git
cd configleam
```

## Usage

Configleam utilizes a Makefile for simplifying various development tasks. Below are the commands available for managing the project's development process:

<details>
<summary>More on Usage</summary>

### Building the Project

To compile the Configleam application and create a binary in the `./build` directory, use:

```bash
make build
```

This command compiles the application, ensuring that any changes to the source code are incorporated into the executable.

### Running the Application

After building, you can run Configleam in development mode with:

```bash
make run
```

This command first builds the project and then executes the compiled binary, starting the application.

### Running Tests

To execute the unit tests for Configleam, ensuring that your changes haven't broken existing functionality, use:

```bash
make test
```

This command runs all unit tests in the project, providing test results for each package.

### Formatting Code

To format the Go source files according to the Go standards, run:

```bash
make fmt
```

This ensures consistency in coding style across the project, making it easier to read and maintain.

### Cleaning Up

To clean up the project, removing build artifacts and clearing the build cache, execute:

```bash
make clean
```

This is useful for ensuring a clean state before a fresh build or after finishing development sessions.

### Getting Help

For a summary of available make commands, you can use:

```bash
make help
```

This will display a list of all commands defined in the Makefile with a brief description of what they do, helping you to quickly find the command you need.

</details>

## Kubernetes Integration

When deployed in a Kubernetes (k8s) environment, Configleam is designed to operate efficiently in a multi-instance setup. This architecture not only boosts availability but also ensures seamless configuration management across instances.

<details>
<summary>More on Kubernetes Integration</summary>

### Multi-Instance Deployment

Configleam can be run in multiple instances within Kubernetes, supporting high availability and scalability. This setup allows for a distributed operation where instances share the load of serving configuration data.

### Leader and Replica Roles

- **Leader Instance:** Among the multiple instances, only the elected leader manages the synchronization with the configuration Git repository. This centralizes the update process, ensuring consistency across configurations.
- **Read Replicas:** Other instances act as read replicas, serving configuration data without performing synchronization tasks. This division of labor ensures efficient resource utilization and quick response times for configuration requests.

### Failover and Leader Election

- **Automatic Failover:** If the current leader instance fails or becomes unavailable, Kubernetes' leader election protocol automatically elects a new leader from the available replicas. This ensures that the synchronization process is always maintained, minimizing downtime and disruption.
- **Seamless Transition:** The newly elected leader initiates the synchronization with the provided Git repositories, ensuring that the latest configurations are fetched and applied. This transition happens automatically, ensuring continuous operation without manual intervention.

### Endpoints for Health and Readiness Checks

- **Health Check Endpoint:** `/health` allows Kubernetes to monitor the overall health of each Configleam instance, facilitating automatic recovery in case of failures.
- **Readiness Check Endpoint:** `/ready` signals to Kubernetes when an instance is ready to serve traffic, ensuring that only fully initialized instances handle requests.

By leveraging Kubernetes' capabilities for leader election and automatic failover, Configleam achieves a resilient and scalable configuration management solution suitable for dynamic cloud-native environments. This setup ensures that configuration synchronization is always active and up-to-date, even in the face of instance failures, providing a robust foundation for microservices architecture.

</details>

## Configuration Repository Structure

The configuration repository is the heart of Configleam, storing all the configuration files needed for your microservices. It is organized in a way that supports multiple environments and flexible configuration declaration.

<details>
<summary>More on Configuration Repository Structure</summary>

### Environment Organization

Configurations are organized by environment, with each environment represented by a separate folder at the root of the repository. For example:

- `/develop`
- `/release`
- `/production`

These folders correspond to the environments in which your microservices will run. The name of each folder could perfectly match the environment variable used when running your microservices.

### Declaring Configuration Variables

Within each environment folder, you can declare your configuration variables in `.yaml` or `.yml` files. These files can be organized as you see fit, including the use of nested folders for additional structure. The key points to remember are:

- **File Format:** Ensure your configuration files are in YAML format, with proper syntax to avoid parsing errors.
- **Flexibility:** You can create as many files as you need, containing as many variables as necessary to suit your configuration requirements.

### Configuration Keys

Configurations are categorized into three types of keys to provide clarity and control over how settings are applied:

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

Here's an example of how these keys might be structured in your YAML files:

```yaml
# Example of global configurations (global.yaml)

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

```yaml
# Example of group configurations with both global and local variables (groups.yaml)

group:analytics:
  - featureFlags # global
  - database: # local
      host: analytics-db-host
      port: 3307
  - additionalMetrics: true # local

group:marketing:
  - database # local
  - featureFlags: # local
      betaFeatures: true
  - marketingCampaignsEnabled: true # local

```

In this example: 

Analytics Group (`group:analytics`): Inherits the global featureFlags and modifies the global database settings for its specific needs. It also includes an analytics-specific setting additionalMetrics.

Marketing Group (`group:marketing:`): Inherits the global database configuration and overrides the featureFlags setting. It introduces a marketing-specific setting marketingCampaignsEnabled.

### Notes

- Global configurations act as default settings. They apply broadly unless overridden by a group-specific configuration.
- Local configurations allow for flexibility and customization within specific groups or contexts.
- Configleam processes these configurations to apply the appropriate settings based on their global or group-specific nature.

</details>

## Security

Configleam implements granular permissions to provide precise control over user actions within Configleam. Each endpoint is guarded by an authorization header ("X-Access-Key"), ensuring that only authenticated users with valid access keys can access the system. Additionally, we've implemented endpoints accessible exclusively for administrators to generate and delete access keys.

<details>
<summary>More on Security</summary>

#### Granular Permissions

Granular permissions are at the core of our security model, allowing us to precisely control user access within Configleam. Each permission corresponds to a specific operation within the system, ensuring that users only have access to the features and functionalities they require.

- **Admin Role:**
  - Description: The admin role grants users global administrative privileges, enabling them to perform all operations across all environments within Configleam.
  - Permissions:
    - `Admin` - Global admin role with access to all operations in all environments.

- **Environment Admin Access:**
  - Description: Similar to the admin role, but restricted to a single environment, providing global administrative privileges within that specific environment.
  - Permissions:
    - `EnvAdminAccess` - Admin role but limited to a single environment.

- **Read Configuration:**
  - Description: Allows users to read configuration settings from Configleam.
  - Permissions:
    - `ReadConfig` - Permission to read configurations.

- **Reveal Secrets:**
  - Description: Grants users the ability to reveal secrets within configuration readings (not yet implemented).
  - Permissions:
    - `RevealSecrets` - Permission to reveal secrets in configurations.

- **Clone Environment:**
  - Description: Permits users to clone existing environments with modifications and delete them.
  - Permissions:
    - `CloneEnvironment` - Permission to clone environments.

- **Create Secrets:**
  - Description: Enables users to create secrets within Configleam.
  - Permissions:
    - `CreateSecrets` - Permission to create secrets.

- **Access Dashboard:**
  - Description: Provides users with access to the dashboard (currently not implemented).
  - Permissions:
    - `AccessDashboard` - Permission to access the dashboard.

#### Access Key Management Endpoints

To facilitate access key management, we've introduced dedicated endpoints that enable users to create and delete access keys securely.

- **Create Access Key:**
  - Endpoint: `POST /access`
  - Description: This endpoint allows users to create or update access keys with specified permissions. Below is an example JSON payload for creating access keys:

```json
{
  "globalAdmin": true,
  "environments": {
    "dev": {
      "envAdminAccess": true,
      "readConfig": true,
      "revealSecrets": false,
      "cloneEnvironment": false,
      "createSecrets": true,
      "accessDashboard": false
    },
    "prod": {
      "envAdminAccess": false,
      "readConfig": true,
      "revealSecrets": false,
      "cloneEnvironment": false,
      "createSecrets": false,
      "accessDashboard": true
    }
  }
}
```

Explanation of JSON properties:
- `globalAdmin`: Boolean indicating whether the access key has global administrative privileges.
- `environments`: Map containing permissions for each environment.
  - `envAdminAccess`: Boolean indicating admin access restricted to the environment.
  - `readConfig`: Boolean indicating permission to read configurations.
  - `revealSecrets`: Boolean indicating permission to reveal secrets.
  - `cloneEnvironment`: Boolean indicating permission to clone environments.
  - `createSecrets`: Boolean indicating permission to create secrets.
  - `accessDashboard`: Boolean indicating permission to access the dashboard.

Response Example:

```json
{
  "globalAdmin": true,
  "environments": {
    "dev": {
      "envAdminAccess": true,
      "readConfig": true,
      "revealSecrets": false,
      "cloneEnvironment": false,
      "createSecrets": true,
      "accessDashboard": false
    },
    "prod": {
      "envAdminAccess": false,
      "readConfig": true,
      "revealSecrets": false,
      "cloneEnvironment": false,
      "createSecrets": false,
      "accessDashboard": true
    }
  },
  "accessKey": "generated-access-key"
}
```

`accessKey`: The newly generated access key that is associated with the provided permissions.

- **Delete Access Key:**
  - Endpoint: `DELETE /access`
  - Description: This endpoint allows administrators to delete access keys.


</details>

## Contributing

Contributions are welcome! Please see the [Contribution Guidelines](CONTRIBUTING.md) for more information.

## Bug Reports and Feature Requests

Please report any issues or feature requests on the [Issue Tracker](https://github.com/raw-lean/configleam/issues).

## License

Distributed under the MIT License. See `LICENSE` for more information.

## Contact

[LinkedIn](https://www.linkedin.com/in/mykhaylo-gusak/)

[Mykhaylo Gusak] - mykhaylogusak@hotmail.com

Project Link: [https://github.com/raw-leak/configleam](https://github.com/raw-leak/configleam)
