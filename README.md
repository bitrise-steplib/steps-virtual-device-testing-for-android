# Virtual Device Testing for Android

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/steps-virtual-device-testing-for-android?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/steps-virtual-device-testing-for-android/releases)

Run Android UI tests on virtual devices

<details>
<summary>Description</summary>

Run Android UI tests on virtual devices. This Step collects the built APK/AAB file from the `$BITRISE_APK_PATH` and in case of instrumentation tests, the `$BITRISE_TEST_APK_PATH` Environment Variables and uses Firebase Test Lab to run UI tests on them.

The available test types are instrumentation, robo, gameloop.

### Configuring the Step

You can read [our detailed guide about using the Step](https://devcenter.bitrise.io/en/testing/device-testing-for-android.html) with other Bitrise Steps. Here we'll go over the configuration options of the Step.

1. Make sure the **App path** input points to the path of the APK or AAB file of your app. If you use our **Android Build** or **Android Build for UI Testing** Steps, you don't need to change the default value.
1. Add the devices you want the tests to run on in the **Test devices** input.

   You need to set the device ID, the version, the orientation, and the language. Read the input description for more information and available devices.
1. Choose a test type.

   The available options are:
   - instrumentation
   - robo
   - gameloop

For detailed configuration options related to the different test types, please check out the [full guide](https://devcenter.bitrise.io/en/testing/device-testing-for-android.html).

You can also export the results of the Step to the Test reports add-on. All you need to do is to add a **Deploy to Bitrise.io** Step to the end of your Workflow.

### Troubleshooting

If you get the **Build already exists** error, it is because you have more than one instance of the Step in your Workflow. This doesn't work as Bitrise sends the build slug to Firebase and having the Step more than once in the same Workflow results in sending the same build slug multiple times.

### Useful links

- [Device testing for Android](https://devcenter.bitrise.io/en/testing/device-testing-for-android.html)
- [Test reports](https://devcenter.bitrise.io/en/testing/test-reports.html)

### Related Steps

- [Android Build](https://www.bitrise.io/integrations/steps/android-build)
- [Android Build for UI Testing](https://www.bitrise.io/integrations/steps/android-build-for-ui-testing)
- [Deploy to Bitrise.io](https://www.bitrise.io/integrations/steps/deploy-to-bitrise-io)
</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://devcenter.bitrise.io/steps-and-workflows/steps-and-workflows-index/).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `app_path` | The path to the app to test (APK or AAB). By default `android-build` and `android-build-for-ui-testing` Steps export the `BITRISE_APK_PATH` Env Var, so you won't need to change this input. Can specify an APK (`$BITRISE_APK_PATH`) or AAB (Android App Bundle) as input (`$BITRISE_AAB_PATH`).  If nothing is specified then the Step will use a default empty Application APK. This will help the library instrumentation tests as it can be used as a shell where the tests will be running.  |  | `$BITRISE_APK_PATH` |
| `test_devices` | Format: One device configuration per line and the parameters are separated with `,` in the order of: `deviceID,version,language,orientation`  For example: `MediumPhone.arm,33,en,portrait`  `MediumTablet.arm,23,en,landscape`  Available devices and its versions: ``` ┌─────────────────────────┬─────────┬────────────────────────────────────────────────┬─────────┬─────────────┬───────────────────────────────┬────────────────────────┐ │         MODEL_ID        │   MAKE  │                   MODEL_NAME                   │   FORM  │  RESOLUTION │         OS_VERSION_IDS        │          TAGS          │ ├─────────────────────────┼─────────┼────────────────────────────────────────────────┼─────────┼─────────────┼───────────────────────────────┼────────────────────────┤ │ AmatiTvEmulator         │ Google  │ Google TV Amati                                │ VIRTUAL │ 1080 x 1920 │ 29                            │ beta=29, deprecated=29 │ │ AndroidTablet270dpi.arm │ Generic │ Generic 720x1600 Android tablet @ 270dpi (Arm) │ VIRTUAL │ 1600 x 720  │ 30                            │                        │ │ GoogleTvEmulator        │ Google  │ Google TV                                      │ VIRTUAL │  720 x 1280 │ 30                            │ beta=30, deprecated=30 │ │ MediumPhone.arm         │ Generic │ Medium Phone, 6.4in/16cm (Arm)                 │ VIRTUAL │ 2400 x 1080 │ 26,27,28,29,30,31,32,33,34,35 │                        │ │ MediumTablet.arm        │ Generic │ Medium Tablet, 10.05in/25cm (Arm)              │ VIRTUAL │ 2560 x 1600 │ 26,27,28,29,30,31,32,33,34,35 │                        │ │ Pixel2.arm              │ Google  │ Pixel 2 (Arm)                                  │ VIRTUAL │ 1920 x 1080 │ 26,27,28,29,30,31,32,33       │                        │ │ SmallPhone.arm          │ Generic │ Small Phone, 4.65in/12cm (Arm)                 │ VIRTUAL │ 1280 x 720  │ 26,27,28,29,30,31,32,33,34,35 │                        │ └─────────────────────────┴─────────┴────────────────────────────────────────────────┴─────────┴─────────────┴───────────────────────────────┴────────────────────────┘ ```  | required | `MediumPhone.arm,33,en,portrait` |
| `test_type` | The type of your test you want to run on the devices. Find more properties below in the selected test type's group.  | required | `robo` |
| `test_apk_path` | The path to the APK that contains instrumentation tests. To build this, you can run the [Build for UI testing](https://bitrise.io/integrations/steps/android-build-for-ui-testing) Step (before this Step). |  | `$BITRISE_TEST_APK_PATH` |
| `inst_test_runner_class` | The fully-qualified Java class name of the instrumentation test runner (leave empty to use the last name extracted from the APK manifest). |  |  |
| `inst_test_targets` | A list of one or more instrumentation test targets to be run (default: all targets). Each target must be fully qualified with the package name or class name, in one of these formats: - `package package_name` - `class package_name.class_name` - `class package_name.class_name#method_name` For example: `class com.my.company.app.MyTargetClass,class com.my.company.app.MyOtherTargetClass`  |  |  |
| `inst_use_orchestrator` | The option of whether running each test within its own invocation of instrumentation with Android Test Orchestrator or not.  | required | `false` |
| `robo_initial_activity` | The initial activity used to start the app during a robo test. (leave empty to get it extracted from the APK manifest) |  |  |
| `robo_max_depth` | The maximum depth of the traversal stack a robo test can explore. Needs to be at least 2 to make Robo explore the app beyond the first activity(leave empty to use the default value: `50`)  |  |  |
| `robo_max_steps` | The maximum number of steps/actions a robo test can execute(leave empty to use the default value: `no limit`).  |  |  |
| `robo_directives` | To complete text fields in your app, use robo-directives and provide a comma-separated list of key-value pairs, where the key is the Android resource name of the target UI element, and the value is the text string. EditText fields are supported but not text fields in WebView UI elements. For example, you could use the following parameter for custom login: ``` username_resource,username,ENTER_TEXT password_resource,password,ENTER_TEXT loginbtn_resource,,SINGLE_CLICK ``` One directive per line, the parameters are separated with `,` character. For example: `ResourceName,InputText,ActionType`  |  |  |
| `robo_scenario_file` | A path to a JSON file with a sequence of recorded actions Robo should perform before the Robo crawl. |  |  |
| `loop_scenarios` | A list of game-loop scenario numbers which will be run as part of the test (default: all scenarios). A maximum of 1024 scenarios may be specified in one test matrix. Format: int,[int,...] For example: ``` 1,2 ```  |  |  |
| `loop_scenario_labels` | A list of game-loop scenario labels (default: None). Each game-loop scenario may be labeled in the APK manifest file with one or more arbitrary strings, creating logical groupings (e.g. GPU_COMPATIBILITY_TESTS).  |  |  |
| `test_timeout` | Max time a test execution is allowed to run before it is automatically canceled. The default value is 900 (15 min), the maximum is 3600 (60 min).  Duration in seconds with up to nine fractional digits. Example: "3.5".  | required | `900` |
| `num_flaky_test_attempts` | Specifies the number of times a test execution should be reattempted if one or more of its test cases fail for any reason. An execution that initially fails but succeeds on any reattempt is reported as FLAKY. The maximum number of reruns allowed is 10. (Default: 0, which implies no reruns.) | required | `0` |
| `obb_files_list` | A list of one or two Android OBB file names which will be copied to each test device before the tests will run (default: None). Each OBB file name must conform to the format as specified by Android (e.g. [main\|patch].0300110.com.example.android.obb) and will be installed into `[shared-storage]/Android/obb/[package-name]/` on the test device. Files should be seperated by newline. For example: ``` main.0300110.com.example.android.obb patch.0300110.com.example.android.obb ```  |  |  |
| `auto_google_login` | Automatically log into the test device using a preconfigured Google account before beginning the test. | required | `false` |
| `environment_variables` | One variable per line, key and value seperated by `=` For example: ``` coverage=true coverageFile=/sdcard/tempDir/coverage.ec ```  |  |  |
| `directories_to_pull` | A list of paths that will be downloaded from the device's storage after the test is complete.  For example  ``` /sdcard/tempDir1 /data/tempDir2 ```  If `download_test_results` input is set to `false` then these files will be available on the dashboard only. To have them downloaded set that input to `true` as well.  |  |  |
| `download_test_results` | If this input is set to `true` all files generated in the test run and the files you downloaded from the device (if you have set `directories_to_pull` input as well) will be downloaded. Otherwise, no any file will be downloaded.  | required | `false` |
| `use_verbose_log` | If set to `true` will enable verbose level logging.  | required | `false` |
| `apk_path` | Deprecated. Use 'App path' input instead of this one. The path to the APK you want the tests run with. By default `gradle-runner` step exports `BITRISE_APK_PATH` env, so you won't need to change this input.  |  |  |
| `app_package_id` | Deprecated: If not specified will be automatically extracted from the App manifest. The Java package of the application under test.  |  |  |
| `inst_test_package_id` | Deprecated: If not specified will be automatically extracted from the Test App manifest. The Java package name of the instrumentation test.  |  |  |
| `api_base_url` | The URL where test API is accessible.  | required | `https://vdt.bitrise.io/test` |
| `api_token` | The token required to authenticate with the API.  | required, sensitive | `$ADDON_VDTESTING_API_TOKEN` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `VDTESTING_DOWNLOADED_FILES_DIR` | The directory containing the downloaded files if you have set `directories_to_pull` and `download_test_results` inputs above. |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/steps-virtual-device-testing-for-android/pulls) and [issues](https://github.com/bitrise-steplib/steps-virtual-device-testing-for-android/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://devcenter.bitrise.io/bitrise-cli/run-your-first-build/).

Learn more about developing steps:

- [Create your own step](https://devcenter.bitrise.io/contributors/create-your-own-step/)
- [Testing your Step](https://devcenter.bitrise.io/contributors/testing-and-versioning-your-steps/)
