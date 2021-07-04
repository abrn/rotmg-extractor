import logging
import os
import json
import subprocess
import re as regex
import ntpath
import UnityPy
import operator
import threading
import xmltodict
from pathlib import Path
from xml.etree import ElementTree
from PIL import Image

from classes import Constants, logger, IndentFilter
from .Helpers import expand_image, find_path, fix_xml, merge_xml, outline_image, parse_int, read_file, read_json, remove_transparency, scale_image, strip_non_alphabetic, write_file


def extract_unity_assets(input_dir, output_path):

    file_patterns = [
        "^globalgamemanagers",
        "^level[0-9]",
        "^resources",
        "^sharedassets[0-9]"
    ]

    ignored_exts = [
        ".resS",
        ".resource"
    ]

    logger.log(logging.INFO, "Extracting build assets...")
    IndentFilter.level += 1

    # Get the _Data directory (where the unity files are located)
    data_dir = find_path(input_dir, "*_Data")

    # Iterate files
    for file_name in os.listdir(data_dir):
        file_path = os.path.join(data_dir, file_name)

        if not os.path.isfile(file_path):
            continue

        continue_loop = False

        for pattern in file_patterns:
            pattern = regex.compile(pattern)
            if pattern.search(file_name):
                continue_loop = True
                break

        for ext in ignored_exts:
            if file_name.endswith(ext):
                continue_loop = False
                break

        if not continue_loop:
            continue

        extract_assets(file_path, output_path)

    IndentFilter.level -= 1
    logger.log(logging.INFO, "Build assets extracted!")


def extract_assets(file_path, output_path):

    file_name = Path(file_path).name
    logger.log(logging.INFO, f"Extracting assets from \"{file_name}\"")
    IndentFilter.level += 1

    obj_type_len = 0  # 13
    obj_name_len = 0  # 35
    path_id_len = 0  # 6

    env = UnityPy.load(file_path)
    for obj in env.objects:

        obj_types = ["TextAsset", "Sprite", "Texture2D", "AudioClip", "MonoScript"]
        if obj.type not in obj_types:
            continue

        data = obj.read()
        output_file = ""

        obj_name = data.name
        if obj_name == "":
            obj_name = "Untitled"

        if obj.type == "TextAsset":
            first_line = data.text.partition("\n")[0]

            ext = "txt"
            if first_line.startswith("<!DOCTYPE html>"):
                ext = "html"
            elif first_line.startswith("<") or "xml" in first_line:
                ext = "xml"
            elif first_line.startswith("{") or first_line.startswith("["):
                ext = "json"

            output_file = output_path / str(obj.type) / f"{obj_name}.{ext}"
            write_file(output_file, data.m_Script, "wb")

        elif obj.type == "Sprite" or obj.type == "Texture2D":
            # print pathid or something like that here
            output_file = output_path / str(obj.type) / f"{obj_name}.png"
            Path(output_file).parent.mkdir(parents=True, exist_ok=True)

            try:
                data.image.save(output_file)
            except Exception as e:
                logger.log(logging.ERROR, f"Error saving {str(obj.type)} \"{obj_name}\" (Path ID: {obj.path_id} in {file_name}) Error: {e}")

        elif obj.type == "AudioClip":
            for name, data in data.samples.items():
                output_file = output_path / str(obj.type) / name
                write_file(output_file, data, "wb")

        elif obj.type == "MonoScript":

            dirs = data.m_Namespace.split(".")
            dirs = [regex.sub('[*?:"<>|]', "", dir) for dir in dirs] # remove invalid file characters
            dir = "/".join(dirs)

            output_file = output_path / str(obj.type) / dir / f"{obj_name}.json"

            keys = ["m_AssemblyName", "m_Namespace", "m_ClassName", "name"]
            base = { key: data.__dict__[key] for key in keys}

            json_pretty = json.dumps(base, indent=4)

            write_file(output_file, json_pretty, "w")


        if output_file != "":

            # logger.log(logging.INFO, f"{str(obj.type)} {obj_name} {obj.path_id} {file_name}")
            # logger.log(logging.INFO, f"{str(obj.type):<13} {obj_name:<35} {obj.path_id:<6} {file_name}")

            if len(str(obj.type)) > obj_type_len:
                obj_type_len = len(str(obj.type)) + 1
            if len(obj_name) > obj_name_len:
                obj_name_len = len(obj_name) + 1
            if len(str(obj.path_id)) > path_id_len:
                path_id_len = len(str(obj.path_id)) + 2

            logger.log(logging.INFO, "{:<{}} {:<{}} {:<{}}".format(
                obj_name, obj_name_len,
                str(obj.type), obj_type_len,
                f"(Path ID: {obj.path_id})", path_id_len
            ))

    IndentFilter.level -= 1


def extract_exalt_version(metadata_file: Path, output_file: Path):
    """ Attempts to find the current version string (e.g. `1.3.2.0.0`) located in `global-metadata.dat` """

    # TODO: Decode/decrypt build version from appsettings

    # A simple regex to capture "1.3.2.0.0" isn't as simple as there are many
    # strings that match. However, the current exalt version is stored in the
    # client as a const string (so it appears in the metadata). It's located
    # in the static class KFFELHLKACG.AFOGMBOANMH.
    # Because it is stored in the metadata, we can use regex to match the
    # string using the previous const strings in the class to get the correct
    # one. (Which is 127.0.0.1 - see the class KFFELHLKACG)

    # For testing:
    # cat global-metadata.dat | grep --text -Po "127\.0\.0\.1[\x00-\x20]*(\d(?:\.\d){4})"

    logger.log(logging.INFO, "Attempting to extract Exalt version string")
    IndentFilter.level += 1

    pattern = regex.compile(b"127\.0\.0\.1[\x00-\x20]*(\d(?:\.\d){4})")

    version_string = ""
    with open(metadata_file, "rb") as file:
        data = file.read()
        result = pattern.findall(data)

        if len(result) == 1:
            version_string = result[0].decode("utf-8")
            logger.log(logging.INFO, f"Exalt version is \"{version_string}\"")
            write_file(output_file, version_string)
        else:
            logger.log(logging.INFO, "Could not extract version string! Must be manually updated.")
            write_file(output_file, "")

    IndentFilter.level -= 1
    return version_string


def merge_manifest_files(manifest_file: Path, input_dir: Path, output_dir: Path):
    logger.log(logging.INFO, f"Merging manifest file...")
    IndentFilter.level += 1

    if not manifest_file.exists():
        logger.log(logging.ERROR, f"Unable to find {manifest_file} !")
        IndentFilter.level -= 1
        return

    manifest = read_json(manifest_file)
    for output_file_name in manifest:
        xml_files = []
        file_names = []
        for merge_file in manifest[output_file_name]:
            if not isinstance(merge_file, dict):
                continue

            merge_file_path = merge_file.get("path")
            if not merge_file_path:
                continue

            file_name = ntpath.basename(merge_file_path)
            if not file_name.endswith("xml"):
                continue

            file_path = input_dir / "TextAsset" / file_name
            if not os.path.isfile(file_path):
                logger.log(logging.ERROR, f"Could not find {file_path} !")
                continue

            xml_files.append(file_path)
            file_names.append(file_name)

        if len(xml_files) == 0:
            continue

        logger.log(logging.DEBUG, f"Merging {len(xml_files)} files. {file_names}")
        merged = merge_xml(xml_files)

        output_file = output_dir / "xml" / f"{output_file_name}.xml"
        write_file(output_file, merged, overwrite=False)
        logger.log(logging.INFO, f"Successfully merged {len(file_names)} files into {output_file_name}.xml")

    IndentFilter.level -= 1


def unpack_launcher_assets(launcher_path, output_path):

    unpacker_file = None
    if os.name == "nt":
        unpacker_file = Constants.LAUNCHER_UNPACKER_WINDOWS
    elif os.name == "posix":
        unpacker_file = Constants.LAUNCHER_UNPACKER_LINUX
    else:
        return

    logger.log(logging.INFO, "Unpacking launcher assets...")
    IndentFilter.level += 1

    process = subprocess.Popen(
        [unpacker_file, launcher_path, output_path],
        stdin=subprocess.PIPE, # bypass "Press any key to exit..."
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT
    )

    logger.pipe(process.stdout)
    process.wait()

    logger.log(logging.INFO, "Done!")
    IndentFilter.level -= 1


def dump_il2cpp(gameassembly: Path, metadata_file: Path, output_dir: Path):

    dumper_file = None
    if os.name == "nt":
        dumper_file = Constants.IL2CPP_DUMPER_WINDOWS
    elif os.name == "posix":
        dumper_file = Constants.IL2CPP_DUMPER_LINUX
    else:
        return

    logger.log(logging.INFO, "Dumping il2cpp...")
    IndentFilter.level += 1

    output_dir.mkdir(parents=True, exist_ok=True)

    process = subprocess.Popen(
        [
            dumper_file, 
            "--bin", gameassembly, 
            "--metadata", metadata_file,
            "--layout", "class",
            "--select-outputs",
            "--py-out",   output_dir / "il2cpp.py",
            "--json-out", output_dir / "metadata.json",
            "--cs-out",   output_dir / "types",
            "--cpp-out",  output_dir / "cpp",
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT
    )

    logger.pipe(process.stdout)
    process.wait()

    logger.log(logging.INFO, "Done!")
    IndentFilter.level -= 1

def extract_sprite_from_spritesheet(spritesheet_image: Path, pos):
    left    = pos["x"]
    top     = pos["y"]
    right   = pos["x"] + pos["w"]
    bottom  = pos["y"] + pos["h"]
    
    with Image.open(spritesheet_image) as img:
        img = img.crop((left, top, right, bottom))
        return img


def extract_texture(sprite_output: Path, sprite_index, sprite_file, spritesheet_json, spritesheet_image, scale=1):
    sprite_output.parent.mkdir(parents=True, exist_ok=True)
    sprite_index = parse_int(sprite_index) # ensure index is an integer, some indexes are stored as a hexadecimal value

    for sprite in spritesheet_json["sprites"]:
        if sprite["index"] == sprite_index and sprite["spriteSheetName"] == sprite_file:
            # found_sprites.append(sprite)
            img = extract_sprite_from_spritesheet(spritesheet_image, sprite["position"])
            img = scale_image(img, scale)
            img.save(sprite_output.with_suffix(".png"))
            return


def extract_animated_texture(sprite_output: Path, sprite_index, spritesheet_name, spritesheet_json, spritesheet_image, generate_gif=True, scale=1):

    sprite_output.parent.mkdir(parents=True, exist_ok=True)
    sprite_index = parse_int(sprite_index) # ensure index is an integer, some indexes are stored as a hexadecimal value
    found_sprites = []

    for sprite in spritesheet_json["animatedSprites"]:
        if sprite["index"] == sprite_index and sprite["spriteData"]["spriteSheetName"] == spritesheet_name:
            found_sprites.append(sprite)

    # sort sprites by direction then action
    # found_sprites.sort(key=operator.itemgetter("direction", "action"))
    found_sprites.sort(key=operator.itemgetter("action", "direction"))
    outline_width = 1

    # Save .png
    chosen_img = extract_sprite_from_spritesheet(spritesheet_image, found_sprites[0]["spriteData"]["position"])
    chosen_img = scale_image(chosen_img, scale)
    # chosen_img = expand_image(chosen_img, outline_width)
    # chosen_img = outline_image(chosen_img, outline_width)
    chosen_img.save(sprite_output.with_suffix(".png"))

    if not generate_gif:
        return

    # each action+direction has multiple frames (usually 2-3).
    # we have sorted the sprites by direction (looks better as a GIF)

    # to actually make the GIF we need a list of PIL.Images
    # to make the gif longer, loop each action+dir a few times
    # then we combine all images into one gif.

    gif_images: list[Image.Image] = []
    gif_frame_timing = 225 # ms of each gif frame
    gif_action_loop = 4 # number of times to repeat the action+dir

    temp_gif_sprites = []
    for i in range(len(found_sprites)):

        sprite = found_sprites[i]
        current_action_dir = (sprite["action"], sprite["direction"])

        stop=False
        if i+1 >= len(found_sprites):
            stop = True
        else:
            next_action_dir = (found_sprites[i+1]["action"], found_sprites[i+1]["direction"])
            stop = current_action_dir != next_action_dir

        temp_gif_sprites.append(sprite)

        if stop:
            # reached last sprite in this action=dir
            # we can now append the gif's frames

            # extract images
            action_imgs = []
            for sprite in temp_gif_sprites:
                img = extract_sprite_from_spritesheet(spritesheet_image, sprite["spriteData"]["position"])
                img = scale_image(img, scale)
                img = expand_image(img, outline_width)
                # img = outline_image(img, outline_width)
                img = remove_transparency(img)
                action_imgs.append(img)
                # gif_images.append(img)

            # append frames
            for i in range(gif_action_loop):
                for img in action_imgs:
                    gif_images.append(img)

            temp_gif_sprites = []

    gif_images[0].save(sprite_output.with_suffix(".gif"), format="GIF", save_all=True, append_images=gif_images[1:], duration=gif_frame_timing, loop=0)


def extract_sprites(output_dir: Path, extracted_assets_dir: Path):

    """
    Just parse all xml files, dump all sprites
    We can do specific functions based on xml boolean tags
    But that can come l8r

    Do this multithreaded ofc
    """

    IMAGE_UPSCALE = 1 #16
    GENERATE_GIF = False

    logger.log(logging.INFO, "Extracting sprites")
    IndentFilter.level += 1

    # file paths
    spritesheet_json = read_json(extracted_assets_dir / "TextAsset" / "spritesheet.json")
    spritesheet_img_animated = extracted_assets_dir / "Texture2D" / "characters.png" # animated textures
    spritesheet_img          = extracted_assets_dir / "Texture2D" / "mapObjects.png" # non animated textures

    xml_file_list = list((extracted_assets_dir / "TextAsset").glob("*.xml"))
    # xml_file_list: list[Path] = [
    #     extracted_assets_dir / "TextAsset" / "belladonna.xml"
    # ]

    # non rotmg xml files
    ignored_files = [
        "assets_manifest.xml",
        "iso_4217.xml"
    ]

    # Iterate all xml files
    for xml_file in xml_file_list:

        if xml_file.name in ignored_files: continue

        json_list = []
        threads: list[threading.Thread] = []

        logger.log(logging.INFO, f"Extracting sprites from \"{xml_file}\"")
        IndentFilter.level += 1
        
        # Iterate all sprites
        tree = ElementTree.fromstring(fix_xml(read_file(xml_file)))
        for i, element in enumerate(tree):
            
            sprite_name = element.get("id")

            # Skip useless/unused sprites
            if sprite_name is None: continue
            if xml_file.name == "pets.xml" and element.find("PetSkin") is None: continue
            if xml_file.name == "equip.xml" and element.tag == "EquipmentSet": continue

            json_obj = xmltodict.parse(ElementTree.tostring(element))
            json_obj = json_obj[element.tag]

            is_animated = element.find("AnimatedTexture") is not None
            is_texture = element.find("Texture") is not None

            sprite_output: Path = None

            # Extract sprite images
            if is_animated:
                sprite_index = element.find("AnimatedTexture/Index").text
                spritesheet_name = element.find("AnimatedTexture/File").text
                sprite_output = output_dir / spritesheet_name / strip_non_alphabetic(sprite_name)

                logger.log(logging.INFO, f"({i+1}/{len(tree)}) Found animated sprite \"{sprite_name}\" [{spritesheet_name}-{sprite_index}]")
                # extract_animated_texture(sprite_output, sprite_index, spritesheet_name, spritesheet_json, spritesheet_img_animated, False, IMAGE_UPSCALE)
                thread = threading.Thread(target=extract_animated_texture, args=(sprite_output, sprite_index, spritesheet_name, spritesheet_json, spritesheet_img_animated, GENERATE_GIF, IMAGE_UPSCALE))
                threads.append(thread)

            elif is_texture:
                sprite_index = element.find("Texture/Index").text
                spritesheet_name = element.find("Texture/File").text
                sprite_output = output_dir / spritesheet_name / strip_non_alphabetic(sprite_name)

                logger.log(logging.INFO, f"({i+1}/{len(tree)}) Found non-animated sprite \"{sprite_name}\" [{spritesheet_name}-{sprite_index}]")
                # extract_texture(sprite_output, sprite_index, spritesheet_name, spritesheet_json, spritesheet_img, IMAGE_UPSCALE)
                thread = threading.Thread(target=extract_texture, args=(sprite_output, sprite_index, spritesheet_name, spritesheet_json, spritesheet_img, IMAGE_UPSCALE))
                threads.append(thread)

            if sprite_output is None:
                logger.log(logging.ERROR, f"Unable to find Texture for sprite \"{sprite_name}\" in {xml_file.name}")
                continue

            # Add json object to list
            url_path = (extracted_assets_dir.parent / sprite_output).relative_to(Constants.WORK_DIR)
            url_path = str(url_path).replace("\\", "/")

            if is_animated:
                del json_obj["AnimatedTexture"]
                json_obj["animated_texture"] = Constants.WEBSERVER_URL + url_path + ".gif"

            elif is_texture:
                del json_obj["Texture"]

            json_obj["texture"] = Constants.WEBSERVER_URL + url_path + ".png"
            json_list.append(json_obj)

        # Do threads
        logger.log(logging.INFO, f"Extracting {len(threads)} sprites")
        for thread in threads: # start all threads
            thread.start()
        for thread in threads: # wait for all threads to complete
            thread.join()

        # Write json file of all extracted sprites
        output_json_file = output_dir / (xml_file.stem + ".json")
        logger.log(logging.INFO, f"Writing to {output_json_file}")
        write_file(output_json_file, json.dumps(json_list, indent=2), overwrite=True)

        logger.log(logging.INFO, f"Done")
        IndentFilter.level -= 1

    IndentFilter.level -= 1
    logger.log(logging.INFO, "Done")
