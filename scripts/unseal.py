from start import read_vaults_from_file

vaults = read_vaults_from_file()
for v in vaults:
    v.wait()
    v.unseal()