# RotMG Resource Extractor

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
                    build_assets.zip
                    unity_assets/
                    xml/
                launcher/
                    build_hash.txt
                    build_assets.zip
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