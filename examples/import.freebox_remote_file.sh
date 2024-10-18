# -------------------------------------------- 👇 is the ID of the task
terraform import "freebox_remote_file.example" 42

# -------------------------------------------- 👇 is the path on the freebox disk
terraform import "freebox_remote_file.example" the-disk/path/to/the/file.txt
