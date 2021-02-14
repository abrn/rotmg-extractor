# RotMG Resource Extractor

Directory Structure

```
src
	temp
		current #contains output from files, before publishing on webserver
			log.txt
			
			production
				client
				launcher
			testing
				client
				launcher
			
			
		files # contains files downloaded from the cdn
			production
				client
				launcher
			testing
				client
				launcher


	output # published on the webserver
	
		production
			client
				build_hash.txt
				build_version.txt
				build.zip
				assets/TextAsset
			launcher
				build_hash.txt
		testing
	
		copy temp/current
		to output/current and output/build_hash
```