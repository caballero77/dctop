
<p align="center"><img src="https://github.com/caballero77/dctop/blob/main/dctop.png" alt="MarineGEO circle logo" style="width: 280px;"/></p>

# dctop

Docker Compose resource monitor that outputs containers cpu, memory, network, io usage and top processes.


## Features

- Showing detailed stats of selected container
- Showing processed and logs of selected container 
- Allows to call up and down commands on selected compose file
- Ability to stop/start and pause/unpause created containers
- Responsive and fast UI with elements selection and scrolling


## Themes

Now dctop only supports [nord](https://www.nordtheme.com/), but I'm going to add a few new themes.

All themes after installation is going to placed in `/usr/local/share/dctop/themes` folder.

You can them in [themes](https://github.com/caballero77/dctop/tree/main/themes) folder. Also feel free to contribute with new themes or create an issue requesting new theme.
## Screenshots

***UI showing running containers, stats, processes and compose file***
![alt text](https://github.com/caballero77/dctop/blob/main/images/normal.png)

***UI before calling up command or after down, just compose file shown***
![alt text](https://github.com/caballero77/dctop/blob/main/images/empty.png)

***UI showing logs of selected container from stdout***
![alt text](https://github.com/caballero77/dctop/blob/main/images/logs.png)

***UI showing compose file with all containers stoped***
![alt text](https://github.com/caballero77/dctop/blob/main/images/stoped.png)

***UI showing compose file with all containers paused***
![alt text](https://github.com/caballero77/dctop/blob/main/images/paused.png)


## License

[GNU v3.0](https://github.com/caballero77/dctop/blob/main/LICENSE)

