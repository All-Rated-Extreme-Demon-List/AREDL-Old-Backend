import dropbox
import os

ACCESS_TOKEN = os.environ.get("DROPBOX_ACCESS_TOKEN")

def main():
    dbx = dropbox.Dropbox(ACCESS_TOKEN)
    
    path = "/data.db"
    output_file = "data.db"
    
    with open(output_file, "wb") as f:
        metadata, res = dbx.files_download(path)
        f.write(res.content)
        
if __name__ == "__main__":
    main()