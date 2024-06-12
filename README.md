# Moonlight Display Auto Switch

This is a tool to automatically activate and deactivate main and fake displays when using [Moonlight](https://moonlight-stream.org/) and [Sunshine](https://app.lizardbyte.dev/Sunshine) for desktop streaming. This will work in conjunction with the [IddSampleDriver](https://github.com/ge9/IddSampleDriver) which will create a fake display for you that can render different screen resolutions than your main display. This program uses [MultiMonitorTool](https://www.nirsoft.net/utils/multi_monitor_tool.html) for determining the active displays and switching their state.

## Prerequisites

Make sure to have `MultiMonitorTool.exe` in your `PATH` environment variable. Also the sunshine log file has to be stored at `%ProgramFiles%\Sunshine\config\sunshine.log`.

No other configuration is necessary. Run the command in background.

## What this program does

- It checks if any main display (which is not a fake display) is active or not
- It checks if a fake display (any display from the IddSampleDriver) is active or not
- It checks if Sunshine is currently streaming the desktop

After this it will check:

1. If the main display is active, the fake display is inactive and Sunshine is streaming it will deactivate the main display and activate the fake display
2. If the main display is inactive, the fake display is active and Sunshine is *not* streaming it will activate the main display and deactivate the fake display

This will give active main displays if you are returning to your computer and will deactivate the main display if you are using sunshine

## Remarks

* There is not automated resolution and refresh rate switching yet. Use the Sunshine [Do and Undo Commands](https://docs.lizardbyte.dev/projects/sunshine/en/latest/about/advanced_usage.html#global-prep-cmd) for this