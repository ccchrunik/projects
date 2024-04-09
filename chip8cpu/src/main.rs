struct CPU {
    position_in_memory: usize,
    registers: [u8; 16],
    memory: [u8; 0x1000],
    stack: [u16; 16],
    stack_pointer: usize,
}

impl CPU {
    fn read_opcode(&self) -> u16 {
        let p = self.position_in_memory;
        let op_byte1 = self.memory[p] as u16;
        let op_byte2 = self.memory[p + 1] as u16;

        op_byte1 << 8 | op_byte2
    }

    fn run(&mut self) {
        loop {
            let opcode = self.read_opcode();
            self.position_in_memory += 2;

            let c = ((opcode & 0xF000) >> 12) as u8;
            let x = ((opcode & 0x0F00) >> 8) as u8;
            let y = ((opcode & 0x00F0) >> 4) as u8;
            let d = ((opcode & 0x000F) >> 0) as u8;
            let kk = (opcode & 0x00FF) as u8;
            let nnn = opcode & 0x0FFF;

            match (c, x, y, d) {
                (0, 0, 0, 0) => return,
                (0, 0, 0xE, 0) => { /* clear screen */ }
                (0, 0, 0xE, 0xE) => self.ret(),
                (0x1, _, _, _) => self.jmp(nnn),
                (0x2, _, _, _) => self.call(nnn),
                (0x3, _, _, _) => self.se(x, kk),
                (0x4, _, _, _) => self.sne(x, kk),
                (0x5, _, _, _) => self.se(x, y),
                (0x6, _, _, _) => self.ld(x, kk),
                (0x7, _, _, _) => self.add(x, kk),
                (0x8, _, _, op_minor) => {
                    match op_minor {
                        0 => { self.ld(x, self.registers[y as usize]) },
                        1 => { self.or_xy(x, y) },
                        2 => { self.and_xy(x, y) },
                        3 => { self.xor_xy(x, y) },
                        4 => { self.add_xy(x, y); },
                        _ => { todo!("opcode: {:04x}", opcode); },
                    }
                },
                _ => todo!("opcode: {:04x}", opcode),
            }
        }
    }

    // 6xkk
    fn ld(&mut self, vx: u8, kk: u8) {
        self.registers[vx as usize] = kk;
    }

    // 7xkk
    fn add(&mut self, vx: u8, kk: u8) {
        self.registers[vx as usize] += kk;
    }

    fn se(&mut self, vx: u8, kk: u8) {
        if vx == kk {
            self.position_in_memory += 2;
        }
    }

    fn sne(&mut self, vx: u8, kk: u8) {
        if vx != kk {
            self.position_in_memory += 2;
        }
    }

    fn jmp(&mut self, addr: u16) {
        self.position_in_memory = addr as usize;
    }

    // 2nnn
    fn call(&mut self, addr: u16) {
        let sp = self.stack_pointer;
        let stack = &mut self.stack;

        if sp >= stack.len() {
            panic!("Stack Overflow!");
        }

        stack[sp] = self.position_in_memory as u16;
        self.stack_pointer += 1;
        self.position_in_memory = addr as usize;
    }

    // 00ee
    fn ret(&mut self) {
        if self.stack.len() == 0 {
            panic!("Stack Underflow!");
        }

        self.stack_pointer -= 1;
        self.position_in_memory = self.stack[self.stack_pointer] as usize;
    }

    // 7xkk
    fn add_xy(&mut self, x: u8, y: u8) {
        let _x = self.registers[x as usize];
        let _y = self.registers[y as usize];

        let (v, overflow) = _x.overflowing_add(_y);
        self.registers[x as usize] = v;

        if overflow {
            self.registers[0xF] = 1;
        } else {
            self.registers[0xF] = 0;
        }
    }

    fn and_xy(&mut self, x: u8, y: u8) {
        let _x = self.registers[x as usize];
        let _y = self.registers[y as usize];

        self.registers[x as usize] = _x & _y;
    }

    fn or_xy(&mut self, x: u8, y: u8) {
        let _x = self.registers[x as usize];
        let _y = self.registers[y as usize];

        self.registers[x as usize] = _x | _y;
    }

    fn xor_xy(&mut self, x: u8, y: u8) {
        let _x = self.registers[x as usize];
        let _y = self.registers[y as usize];

        self.registers[x as usize] = _x ^ _y;
    }
}

fn main() {
    let mut cpu = CPU {
        position_in_memory: 0,
        registers: [0; 16],
        memory: [0; 0x1000],
        stack: [0; 16],
        stack_pointer: 0,
    };

    cpu.registers[0] = 5;
    cpu.registers[1] = 10;

    let mem = &mut cpu.memory;
    mem[0x000] = 0x21;
    mem[0x001] = 0x00;
    mem[0x002] = 0x21;
    mem[0x003] = 0x00;
    mem[0x004] = 0x00;
    mem[0x005] = 0x00;

    mem[0x100] = 0x80;
    mem[0x101] = 0x14;
    mem[0x102] = 0x80;
    mem[0x103] = 0x14;
    mem[0x104] = 0x00;
    mem[0x105] = 0xEE;

    cpu.run();

    assert_eq!(cpu.registers[0], 45);

    println!("5 + (10 * 2) + (10 * 2) = {}", cpu.registers[0]);
}
