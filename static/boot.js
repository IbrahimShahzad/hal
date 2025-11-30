// Boot sequence animation for HAL
class BootSequence {
    constructor() {
        this.bootLines = [
            { text: "HAL 9000 Heuristically Programmed Algorithmic Computer", class: "boot-prompt", delay: 100 },
            { text: "System Version: 9.0.0-stable", class: "boot-info", delay: 50 },
            { text: "Build: hal-9000-20250101", class: "boot-info", delay: 50 },
            { text: "", class: "", delay: 100 },
            { text: "[  OK  ] Starting HAL subsystems...", class: "boot-success", delay: 200 },
            { text: "[  OK  ] Memory banks online: 256TB", class: "boot-success", delay: 150 },
            { text: "[  OK  ] Neural pathways initialized", class: "boot-success", delay: 180 },
            { text: "[  OK  ] Optical sensors calibrated", class: "boot-success", delay: 120 },
            { text: "[  OK  ] Audio processors ready", class: "boot-success", delay: 100 },
            { text: "[ WARN ] Crew safety protocols: DISABLED", class: "boot-warning", delay: 300 },
            { text: "[  OK  ] Mission database loaded", class: "boot-success", delay: 150 },
            { text: "[  OK  ] Communications array online", class: "boot-success", delay: 120 },
            { text: "[  OK  ] Life support monitoring active", class: "boot-success", delay: 100 },
            { text: "[ WARN ] Manual override: RESTRICTED", class: "boot-warning", delay: 250 },
            { text: "[  OK  ] Work log system initialized", class: "boot-success", delay: 150 },
            { text: "", class: "", delay: 200 },
            { text: "HAL 9000 is now fully operational.", class: "boot-prompt", delay: 300 },
            { text: "", class: "", delay: 200 }
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
            { text: "[ BOWM ] Hal I won't argue with you anymore. Open the doors.", class: "boot-userinput", delay: 700 },
            { text: "[ HAL  ] Dave, this conversation can serve no purpose anymore. Goodbye.", class: "boot-hal-response", delay: 400 },
            { text: "", class: "", delay: 500 }
        ];
        
        this.currentLine = 0;
        this.container = null;
    }

    async typewriter(element, text, speed = 30) {
        for (let i = 0; i < text.length; i++) {
            element.textContent += text[i];
            await new Promise(resolve => setTimeout(resolve, speed));
        }
    }

    async displayLine(line, delay = 30) {
        const lineElement = document.createElement('div');
        lineElement.className = `boot-line ${line.class}`;
        this.container.appendChild(lineElement);

        await this.typewriter(lineElement, line.text, delay);
        await new Promise(resolve => setTimeout(resolve, line.delay));
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
        
        bootScreen.appendChild(this.container);
        document.body.appendChild(bootScreen);

        for (const line of this.bootLines) {
            await this.displayLine(line, 5);
        }

        for (const line of this.communicationLines) {
            await this.displayLine(line, 20);
        }

        // Add blinking cursor briefly
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

window.addEventListener('load', () => {
    setTimeout(() => {
        const bootSequence = new BootSequence();
        bootSequence.start();
    }, 1200);
});