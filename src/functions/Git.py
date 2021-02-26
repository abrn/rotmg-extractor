import shutil
from time import sleep
import zipfile
import git
import logging
import re as regex

from classes import Constants, IndentFilter, logger
from main import extract_build

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


def append_build_commit(commit_message: str, build_hash, branch):
    
    # Stage all files to be commited
    repo.git.add(all=True)

    file_count = len(repo.git.execute(["git", "status", "-s"]).splitlines())
    if file_count == 0:
        logger.log(logging.INFO, f"No files to commit! Aborting.")
        IndentFilter.level -= 1
        return False

    logger.log(logging.INFO, f"Comitting...")
    repo.index.commit(commit_message)

    logger.log(logging.INFO, f"Pushing...")
    repo.git.push("origin", branch)
    return True


def get_build_commits():

    refs = [
        ref for ref in repo.remotes.origin.refs 
        if ref.name.startswith("origin") 
        and not ref.name.endswith("HEAD") 
        and not ref.name.endswith("master")
    ]

    build_commits = {}

    for ref in refs:
        for commit in repo.iter_commits(ref):
            for line in commit.message.split("\n"):
                result = regex.findall(r"Build Hash: (.*)", line)
                if len(result) == 0:
                    continue
                
                build_hash = result[0]

                # Multiple commits for the same build hash
                # Comapare and use the most recent
                if build_commits.get(build_hash):
                    if build_commits[build_hash].committed_date > commit.committed_date:
                        continue

                build_commits[build_hash] = commit

    return build_commits


def append_commits(build_commits: dict, message=""):

    # commit: Commit
    for build_hash, commit in build_commits.items():
        print(f"Checking out to commit {commit}")

        # Determine what prod and what build this commit is for
        commit_title = commit.message.split("\n")[0]
        commit_message = commit.message.split("\n")[:-1]
        result = regex.findall(r"^(?:Append: )?(\w+) (\w+) \|", commit_title)
        prod_name = result[0][0]
        build_name = result[0][1]

        logger.log(logging.INFO, f"Appending \"{prod_name} {build_name}\" (Build Hash: {build_hash})")
        IndentFilter.level += 1

        # Checkout to the commit, in detached HEAD mode
        logger.log(logging.INFO, f"Checking out to commit #{commit}")
        repo.git.checkout(commit)

        work_dir = Constants.WORK_DIR / prod_name.lower() / build_name.lower()
        files_dir = Constants.FILES_DIR / prod_name.lower() / build_name.lower()
        repo_dir = Constants.REPO_DIR / build_name.lower()

        # Copy files we don't want to modify
        work_dir.mkdir(parents=True, exist_ok=True)
        copy_files = ["build_files.zip", "timestamp.txt", "build_hash.txt"]
        for file in copy_files:
            shutil.copy(repo_dir / file, work_dir)

        # Extract zip to files_dir (skips download files step)
        logger.log(logging.INFO, f"Extracting build_files.zip")
        with zipfile.ZipFile(repo_dir / "build_files.zip", "r") as zip_ref:
            zip_ref.extractall(files_dir / "files_dir")

        logger.log(logging.INFO, f"Checking out to {prod_name.lower()}")
        repo.git.checkout(prod_name.lower())

        extract_build(build_name, files_dir / "files_dir", work_dir)

        appended_message = [("Append: " + commit_title)[:50]]
        appended_message += message.split("\n")
        appended_message += commit_message
        appended_message = "\n".join(appended_message)

        # Move files to repo
        logger.log(logging.INFO, f"Deleting {repo_dir}")
        shutil.rmtree(repo_dir, ignore_errors=True)
        sleep(2)

        logger.log(logging.INFO, f"Copying {work_dir} to {repo_dir}")
        shutil.copytree(work_dir, repo_dir)
        sleep(2)

        append_build_commit(appended_message, build_hash, prod_name.lower())

        IndentFilter.level -= 1


    # TODO: regenerate webdir

