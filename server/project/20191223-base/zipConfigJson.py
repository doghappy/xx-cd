import os, zipfile


def make_zip(source_dir, output_filename):
    zipf = zipfile.ZipFile(output_filename, 'w',zipfile.ZIP_DEFLATED)    
    pre_len = len(os.path.dirname(source_dir))
    for parent, dirnames, filenames in os.walk(source_dir):
        for filename in filenames:
            pathfile = os.path.join(parent, filename)
            if pathfile.find(".meta") <= 0:
                arcname = pathfile[pre_len:].strip(os.path.sep)
                zipf.write(pathfile, arcname)
    zipf.close()
 
 
if __name__ == '__main__':
    make_zip('config', 'D:/build/project/20191223-base/assets/localBundle/config.bin')