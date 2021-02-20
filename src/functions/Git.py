import git
import logging
from classes import Constants
from classes.CustomLogger import IndentFilter, logger

repo = git.Repo.init(Constants.REPO_DIR)

def commit_new_build(prod_name, build_name, app_settings, exalt_version=""):

    logger.log(logging.INFO, f"Comitting files to repository...")
    IndentFilter.level += 1

    commit_name = f"{prod_name} {build_name} | {app_settings['build_hash']}"
    commit_msg_lines = []

    # Include the exalt version in the commit name
    if build_name == "Client":
        if exalt_version != "":
            commit_name = f"{prod_name} {build_name} | {exalt_version} | {app_settings['build_hash']}"
        else:
            commit_msg_lines.append(f"ERROR: Exalt version wasn't extracted - must be manually updated!")

    # Truncate commit name to 50 char limit
    commit_name = commit_name[:50]

    # Stage all files to be commited
    repo.git.add(all=True)

    file_count = len(repo.git.execute(["git", "status", "-s"]).splitlines())
    if file_count == 0:
        logger.log(logging.INFO, f"No files to commit! Aborting.")
        IndentFilter.level -= 1
        return False

    commit_msg_lines.append(f"Build ID: {app_settings['build_id']}")
    commit_msg_lines.append(f"Build Hash: {app_settings['build_hash']}")
    commit_msg_lines.append(f"Build Version: {app_settings['build_version']}")
    # commit_msg_lines.append(f"Build CDN: {app_settings['build_cdn']}")

    commit_message = "\n".join(commit_msg_lines)
    commit_message = commit_name + "\n" + commit_message

    logger.log(logging.INFO, f"Comitting {file_count} changed files...")
    repo.index.commit(commit_message)

    logger.log(logging.INFO, f"Pushing to remote...")
    repo.git.push("origin", prod_name.lower())

    logger.log(logging.INFO, f"Done")
    IndentFilter.level -= 1
    return True