use std::env;
use std::fs;

fn main() {
    // 1. Get arguments (Input Path, Output Path)
    let args: Vec<String> = env::args().collect();
    
    if args.len() < 3 {
        return;
    }

    let input_path = &args[1];
    let output_path = &args[2];

    // 2. Read the file
    let content = fs::read_to_string(input_path).unwrap_or_default();

    // 3. THE LOGIC (Clean the @)
    let clean_content = content.replace("@", "");

    // 4. Write result
    fs::write(output_path, clean_content).unwrap();
}