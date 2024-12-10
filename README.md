# Windows Installation
1. Install Go with the Installer from https://go.dev/dl/go1.23.3.windows-386.msi
2. Clone this repository
> git clone https://github.com/VachellChristiansen/skripsi
3. Change Directory to the cloned repository
> cd skripsi
4. Run the program
> go run .

# Linux Installation (Debian/Ubuntu Based)
1. Download Go Tarball from https://go.dev/dl/go1.23.3.linux-amd64.tar.gz
2. Rmove existing installation (adjust the path if go was installed in a different place)
> sudo rm -rf /usr/local/go
3. Extract Go
> sudo tar -C /usr/local -xzf go1.23.3.linux-amd64.tar.gz
4. Add Go to Environment Path
> echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bashrc
> source ~/.bashrc
5. Clone this repository
> git clone https://github.com/VachellChristiansen/skripsi
6. Change Directory to the cloned repository
> cd skripsi
7. Run the program
> go run .

## Manual Book Available: [Click Here](https://www.figma.com/design/jgFXloSvYLuYV3HRtR0KOY/ManualBookSkripsi?node-id=0-1&t=IVDrHVF4twjehoJ1-1)