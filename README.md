# indexcheck

## About this plugin
This plugin uses the scan status Xray API to determine what files within repositories (or builds) are marked as unscanned. This requires Xray 3.34 and above.

## Installation with JFrog CLI
Since this plugin is currently not included in [JFrog CLI Plugins Registry](https://github.com/jfrog/jfrog-cli-plugins-reg), it needs to be built and installed manually. Follow these steps to install and use this plugin with JFrog CLI.
1. Make sure JFrog CLI is installed on your machine by running ```jfrog```. If it is not installed, [install](https://jfrog.com/getcli/) it.
2. Create a directory named ```plugins``` under ```~/.jfrog/``` if it does not exist already.
3. Clone this repository.
4. CD into the root directory of the cloned project.
5. Run ```go build``` to create the binary in the current directory.
6. Copy the binary into the ```~/.jfrog/plugins``` directory.
7. Get a copy of your supported_types.json (located under ```$ARTIFACTORY_HOME/var/data/artifactory/.cache/xray```) and put it under the CLI HOME ```~/.jfrog/```.

## Usage
### Commands
* check
    - Arguments:
        - repo-all: verify all repositories.
        - build-list: verify comma delimited list of builds
        - build-single: verify a speciic build
        - repo-list: verify comma delimited list of repositories.
        - repo-single: verify a single repository.
        - repo-path: verify a speciic path within a repository.
    - Flags:
        - worker: Worker count for getting scan details **[Default: 1]**
        - showall: Show all results, scanned or not **[Default: false]**
        - reindex: force reindex unscanned artifacts
    - Example:
    ```
   $ jfrog indexcheck check repo-list generic-local,docker-local --showall
   checking:generic-local at path:
   not scanned         	 7.6 kB     	 json                        generic-local:/centos/manifest.json
   not scanned         	 541 B      	 x-gzip                      generic-local:/centos/sha256__07b7d7b4253517e953bd39a5059e16d32afca518bbaa12a466917c0bda7bc5aa.tar.gz
   not scanned         	 572.6 MB   	 x-gzip                      generic-local:/centos/sha256__153c9ca3d7b8383f4dd5b6a4824ce19b68825bbd6a6c7691199435a8f0840eab.tar.gz
   not scanned         	 657 B      	 x-gzip                      generic-local:/centos/sha256__05236417d65b6cbe061c7ed0331bf6975406631a274eab18f66dc9b13d8fdb84.tar.gz
    ```
    ![](demo-check.gif)    
* graph
    - Arguments:
        - none
    - Flags:
        - interval: Polling interval in seconds **[Default: 1]**
        - retry: Show retry queues in chart **[Default: false]**
    - Example:
    ```
   $ jfrog indexcheck graph
    ```
    ![](demo-graph.gif)
* metrics
    - Arguments:
        - list - list metrics
    - Flags:
        - raw: Output straight from Xray **[Default: false]**
        - min: Get minimum JSON from Xray (no whitespace) **[Default: false]**
    - Example:
    ```
  $ jfrog indexcheck metrics --min
  [{"name":"sys_memory_used_bytes","help":"Host used virtual memory","type":"GAUGE","metrics":[{"timestamp_ms":"1638862071581","value":"1.9554074624e+10"}]},{"name":"app_self_metrics_total","help":"Count of collected metrics","type":"GAUGE","metrics":[{"timestamp_ms":"1638862071581","value":"35"}]},{"name":"jfxr_data_artifacts_total","help":"Artifacts of pkg type npm count in Xray","type":"COUNTER","metrics":[{"labels":{"package_type":"build"},"timestamp_ms":"1638862071581","value":"628"},{"labels":{"package_type":"deb"},"timestamp_ms":"1638862071581","value":"16"},{"labels":{"package_type":"docker"},"timestamp_ms":"1638862071581","value":"486"},{"labels":{"package_type":"generic"},"timestamp_ms":"1638862071581","value":"239"},{"labels":{"package_type":"go"}
  ```
### Environment variables
JFROG_CLI_LOG_LEVEL This variable determines the log level of the JFrog CLI.
Possible values are: INFO, ERROR, and DEBUG.
If set to ERROR, JFrog CLI logs error messages only. It is useful when you wish to read or parse the JFrog CLI output and do not want any other information logged.

## Additional info
None.

## Release Notes
The release notes are available [here](RELEASE.md).
