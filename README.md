# RotMG Resource Extractor

## TODO
Create diff:
  Either generate a static html/txt file
  or use a git repository (unlikely, we ~1.25gb of files every release, that
  shit will build up hellllla quick)

When download a URL apply use the rename_duplicate_file function

When downloading all client assets, preserve the original directory structure

Append to main to autoloop the function (instead of crontab)
We do this so we don't accidentally run the script while it's already running
(in the middle of extracting a build)
```
sleep(5 * 60)
main()
```

## Directory Structure

```
src/
    temp/
        current/
            log.txt
            {prod_name}/
                app_settings.xml
                client/
                    build_hash.txt
                    exalt_version.txt
                    unity_assets.zip
                    unity_assets/
                    xml/
                launcher/
                    build_hash.txt
                    unity_assets.zip
                    unity_assets/

        files/
            {prod_name}/
                client/
                launcher/
                    Installer.exe
                    programfiles/

        output/
            README.txt
            last_updated.txt
            current/
            {build_hash}/
                {prod_name}/
                    client/
                    launcher/
```
Directory meanings:
```
src/temp            -- Temp directory cleared everytime the program is run
src/temp/current    -- Work directory to extract everything before publishing to ouput_dir
src/temp/files      -- Temp directory to download files to
src/output          -- Public directory on the webserver
```