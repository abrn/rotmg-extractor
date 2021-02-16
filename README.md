# RotMG Resource Extractor

## Directory Structure

```
output/
  repo/ - current build
    testing/
    production/
      client/
        timestamp.txt
        build_hash.txt
        exalt_version.txt
        build_files.zip
        extracted_assets/
          TextAsset etc
      launcher/
        timestamp.txt
        build_hash.txt
        Rotmg-Exalt-Installer.zip
        build_files.zip
        extracted_assets/

  web/
    README.txt
    last_updated.txt
    testing
    production
      client
      launcher
        current/
        {build_hash}/
          # see repo

  temp/
    files/
      testing/
      production/
        ...
    work
      testing/
      production/
        # see repo
```