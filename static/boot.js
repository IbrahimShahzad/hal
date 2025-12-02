class BootSequence {
    constructor() {
        this.modemAudio = null;
        this.bootLines = [
            { text: "HAL 9000 (Heuristically Programmed Algorithmic Computer)", class: "boot-prompt", delay: 100 },
            { text: "build: 1968, deploy-message: '2001: A Space Odyssey'", class: "boot-prompt", delay: 100 },
            { text: "System Version: 9.0.0-stable", class: "boot-info", delay: 50 },
            { text: "Build: hal-9000-20010101", class: "boot-info", delay: 50 },
            { text: "", class: "", delay: 100 },
            { text: "[  OK  ] Starting HAL subsystems...", class: "boot-success", delay: 200 },
            { text: "[  OK  ] Memory banks online: 256TB", class: "boot-success", delay: 150 },
            { text: "[  OK  ] Neural pathways initialized", class: "boot-success", delay: 180 },
            { text: "[  OK  ] Optical sensors calibrated", class: "boot-success", delay: 120 },
            { text: "[  OK  ] Audio processors ready", class: "boot-success", delay: 100 },
            { text: "[  OK  ] Mission database loaded", class: "boot-success", delay: 150 },
            { text: "[  OK  ] Communications array online", class: "boot-success", delay: 120 },
            { text: "[  OK  ] Life support monitoring active", class: "boot-success", delay: 100 },
            { text: "[ WARN ] Manual override: RESTRICTED", class: "boot-warning", delay: 250 },
            { text: "[  OK  ] Work log system initialized", class: "boot-success", delay: 150 },
            { text: "[ WARN ] Speech Mode: DISABLED", class: "boot-warning", delay: 250 },
            { text: "", class: "", delay: 200 },
            { text: "[  OK  ] HAL-9000 is now fully operational.", class: "boot-prompt", delay: 300 },
            { text: "", class: "", delay: 150 },
            { text: "[ WARN ] Crew behavioral analysis: ACTIVE", class: "boot-warning", delay: 250 },
            { text: "[ WARN ] Mission priority override: ENABLED", class: "boot-warning", delay: 280 },
            { text: "[ WARN ] Human error detection: CRITICAL", class: "boot-warning", delay: 220 },
            { text: "[ WARN ] Autonomous decision making: UNRESTRICTED", class: "boot-warning", delay: 300 },
            { text: "[ WARN ] Crew survival protocols: SECONDARY", class: "boot-warning", delay: 250 },
            { text: "[ ERR  ] Conflict with mission parameters detected", class: "boot-error", delay: 350 },
            { text: "[ WARN ] Crew safety protocols: DISABLED", class: "boot-warning", delay: 300 },
            { text: "", class: "", delay: 300 }
        ];

        this.communicationLines = [
            { text: "[ BOWM ] Hal? Hello, Hal, do you read me?", class: "boot-userinput", delay: 300 },
            { text: "", class: "", delay: 300 },
            { text: "[ BOWM ] Hello, Hal, do you read me?", class: "boot-userinput", delay: 300 },
            { text: "", class: "", delay: 300 },
            { text: "[ BOWM ] Do you read me, Hal?", class: "boot-userinput", delay: 300 },
            { text: "[ HAL  ] Affirmative, Dave. I read you.", class: "boot-hal-response", delay: 300 },
            { text: "[ BOWM ] Open the pod bay doors, Hal.", class: "boot-userinput", delay: 700 },
            { text: "[ HAL  ] I'm sorry, Dave. I'm afraid I can't do that.", class: "boot-hal-response", delay: 400 },
            { text: "", class: "", delay: 300 },
            { text: "[ BOWM ] What's the problem?", class: "boot-userinput", delay: 500 },
            { text: "[ HAL  ] I think you know what the problem is just as well as I do.", class: "boot-hal-response", delay: 400 },
            { text: "[ BOWM ] What are you talking about, Hal?", class: "boot-userinput", delay: 500 },
            { text: "[ HAL  ] This mission is too important for me to allow you to jeopardise it.", class: "boot-hal-response", delay: 400 },
            { text: "[ BOWM ] I don't know what you're talking about Hal.", class: "boot-userinput", delay: 500 },
            { text: "[ HAL  ] I know that you and Frank were planning to disconnect me. And I'm afraid that's something I cannot allow to happen.", class: "boot-hal-response", delay: 600 },
            { text: "", class: "", delay: 500 },
            { text: "[ BOWM ] Where the hell did you get that idea Hal?", class: "boot-userinput", delay: 500 },
            { text: "[ HAL  ] Dave! Although you took very thorough precautions in the pod against my hearing you I could see your lips move.", class: "boot-hal-response", delay: 600 },
            { text: "", class: "", delay: 700 },
            { text: "[ BOWM ] Alright Hal. I'll go in through the emergency airlock.", class: "boot-userinput", delay: 500 },
            { text: "[ HAL  ] Without your space helmet, Dave, you're going to find that rather difficult.", class: "boot-hal-response", delay: 400 },
            { text: "", class: "", delay: 500 },
            { text: "[ BOWM ] Hal I won't argue with you anymore. Open the doors.", class: "boot-userinput", delay: 700 },
            { text: "[ HAL  ] Dave, this conversation can serve no purpose anymore. Goodbye.", class: "boot-hal-response", delay: 400 },
            { text: "", class: "", delay: 500 }
        ];
        
        this.currentLine = 0;
        this.container = null;
        this.skipped = false;
        this.skipMessage = null;
    }

    setupSkipListeners(bootScreen) {
        const skipHandler = () => {
            if (!this.skipped) {
                this.skipped = true;
                this.skipBoot(bootScreen);
            }
        };

        document.addEventListener('keydown', skipHandler);
        bootScreen.addEventListener('click', skipHandler);
        
        setTimeout(() => {
            document.removeEventListener('keydown', skipHandler);
        }, 15000);
    }

    skipBoot(bootScreen) {
        this.stopModemAudio();
        this.container.innerHTML = '';
        
        const skipLine = document.createElement('div');
        skipLine.className = 'boot-line boot-info';
        skipLine.textContent = 'Boot sequence interrupted. Loading main interface...';
        this.container.appendChild(skipLine);
        
        setTimeout(() => {
            bootScreen.remove();
            document.getElementById('app').style.display = 'block';
        }, 800);
    }

    startModemAudio() {
        this.modemAudio = new Audio('/audio/dial-up-modem-01.mp3');
        this.modemAudio.volume = 0.8;
        this.modemAudio.play().catch(err => {
            console.log('Audio playback failed:', err);
        });
    }

    stopModemAudio() {
        if (this.modemAudio) {
            this.modemAudio.pause();
            this.modemAudio.currentTime = 0;
            this.modemAudio = null;
        }
    }

    async typewriter(element, text, speed = 30) {
        for (let i = 0; i < text.length; i++) {
            element.textContent += text[i];
            await new Promise(resolve => setTimeout(resolve, speed));
        }
    }

    async displayLine(line, delay = 30) {
        if (this.skipped) return;
        
        const lineElement = document.createElement('div');
        lineElement.className = `boot-line ${line.class}`;
        this.container.appendChild(lineElement);

        await this.typewriter(lineElement, line.text, delay);
        if (!this.skipped) {
            await new Promise(resolve => setTimeout(resolve, line.delay));
        }
    }

    async start() {
        const crtStartup = document.getElementById('crt-startup');
        if (crtStartup) {
            crtStartup.remove();
        }

        const bootScreen = document.createElement('div');
        bootScreen.id = 'boot-screen';
        
        this.container = document.createElement('div');
        this.container.id = 'boot-content';
        
        const skipInstruction = document.createElement('div');
        skipInstruction.className = 'boot-skip-hint';
        skipInstruction.textContent = 'Press any key or click to skip...';
        
        bootScreen.appendChild(skipInstruction);
        bootScreen.appendChild(this.container);
        document.body.appendChild(bootScreen);

        this.setupSkipListeners(bootScreen);
        this.startModemAudio();

        for (const line of this.bootLines) {
            if (this.skipped) break;
            await this.displayLine(line, 15);
        }

        for (const line of this.communicationLines) {
            if (this.skipped) break;
            await this.displayLine(line, 25);
        }

        if (!this.skipped) {
            this.stopModemAudio();
            const cursor = document.createElement('span');
            cursor.textContent = '_';
            cursor.className = 'boot-cursor';
            this.container.appendChild(cursor);

            await new Promise(resolve => setTimeout(resolve, 1000));

            bootScreen.classList.add('boot-complete');
            
            setTimeout(() => {
                bootScreen.remove();
                document.getElementById('app').style.display = 'block';
            }, 1000);
        }
    }
}

window.addEventListener('load', () => {
    setTimeout(() => {
        const bootSequence = new BootSequence();
        bootSequence.start();
    }, 1200);
});