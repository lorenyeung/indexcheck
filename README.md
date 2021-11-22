# indexcheck

## About this plugin
This plugin uses the.

## Installation with JFrog CLI
Since this plugin is currently not included in [JFrog CLI Plugins Registry](https://github.com/jfrog/jfrog-cli-plugins-reg), it needs to be built and installed manually. Follow these steps to install and use this plugin with JFrog CLI.
1. Make sure JFrog CLI is installed on your machine by running ```jfrog```. If it is not installed, [install](https://jfrog.com/getcli/) it.
2. Create a directory named ```plugins``` under ```~/.jfrog/``` if it does not exist already.
3. Clone this repository.
4. CD into the root directory of the cloned project.
5. Run ```go build``` to create the binary in the current directory.
6. Copy the binary into the ```~/.jfrog/plugins``` directory.

## Usage
### Commands
* <TBD>
    - Arguments:
        - none
    - Flags:
        - interval: Polling interval in seconds **[Default: 1]**
    - Example:
    ```
   $ jfrog indexcheck <TBD>
    ```
### Environment variables
None

## Additional info
None.

## Release Notes
The release notes are available [here](RELEASE.md).
