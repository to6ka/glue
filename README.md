# GoZix Glue

The package represents a very simple and easy implementation of the extensible application on golang. The core 
components of an application are bundles that are glued together using a dependency injection container.
   
## Built-in Tags

| Symbol                 | Value                | Description                      | 
| ---------------------- | -------------------- | -------------------------------- |
| TagCliCommand          | cli.cmd              | Add a cli command                |
| TagRootPersistentFlags | cli.persistent_flags | Add custom flags to root command |

## Built-in Services

| Symbol        | Value           | Description             | 
| ------------- | --------------- | ----------------------- |
| DefCliRoot    | cli.cmd.root    | Add root cli command    |
| DefCliVersion | cli.cmd.version | Add version cli command |
| DefRegistry   | registry        | A key/value registry    |
