use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use std::io::{self, Read};

#[link(wasm_import_module = "gojinn")]
extern "C" {
    fn host_log(level: u32, ptr: u32, len: u32);
    fn host_db_query(q_ptr: u32, q_len: u32, out_ptr: u32, out_max: u32) -> u32;
    fn host_kv_set(k_ptr: u32, k_len: u32, v_ptr: u32, v_len: u32);
    fn host_kv_get(k_ptr: u32, k_len: u32, out_ptr: u32, out_max: u32) -> u64;
    fn host_ask_ai(p_ptr: u32, p_len: u32, out_ptr: u32, out_max: u32) -> u64;
}


pub mod logger {
    use super::*;

    pub fn info(msg: &str) {
        unsafe { host_log(1, msg.as_ptr() as u32, msg.len() as u32) };
    }

    pub fn error(msg: &str) {
        unsafe { host_log(3, msg.as_ptr() as u32, msg.len() as u32) };
    }
}

pub mod db {
    use super::*;

    pub fn query(sql: &str) -> Result<Vec<serde_json::Value>, String> {
        let mut buffer = vec![0u8; 65536]; 
        
        let written = unsafe {
            host_db_query(
                sql.as_ptr() as u32,
                sql.len() as u32,
                buffer.as_mut_ptr() as u32,
                buffer.len() as u32,
            )
        };

        if written == 0 {
            return Err("Query returned empty or failed".to_string());
        }

        let json_slice = &buffer[..written as usize];
        
        match serde_json::from_slice::<Vec<serde_json::Value>>(json_slice) {
            Ok(rows) => {
                if let Some(first) = rows.first() {
                    if let Some(err_msg) = first.get("error") {
                        return Err(err_msg.as_str().unwrap_or("Unknown DB Error").to_string());
                    }
                }
                Ok(rows)
            }
            Err(e) => Err(format!("Failed to parse DB response: {}", e)),
        }
    }
}

pub mod kv {
    use super::*;

    pub fn set(key: &str, value: &str) {
        unsafe {
            host_kv_set(
                key.as_ptr() as u32,
                key.len() as u32,
                value.as_ptr() as u32,
                value.len() as u32,
            )
        };
    }

    pub fn get(key: &str) -> Option<String> {
        let mut buffer = vec![0u8; 4096];
        
        let written = unsafe {
            host_kv_get(
                key.as_ptr() as u32,
                key.len() as u32,
                buffer.as_mut_ptr() as u32,
                buffer.len() as u32,
            )
        };

        if written > buffer.len() as u64 {
            return None;
        }

        let val = String::from_utf8_lossy(&buffer[..written as usize]).to_string();
        Some(val)
    }
}

pub mod ai {
    use super::*;

    pub fn ask(prompt: &str) -> String {
        let mut buffer = vec![0u8; 65536];
        
        let written = unsafe {
            host_ask_ai(
                prompt.as_ptr() as u32,
                prompt.len() as u32,
                buffer.as_mut_ptr() as u32,
                buffer.len() as u32,
            )
        };

        if written == 0 {
            return "AI Error".to_string();
        }

        String::from_utf8_lossy(&buffer[..written as usize]).to_string()
    }
}


#[derive(Deserialize)]
pub struct Request {
    pub body: String,
    pub headers: Option<HashMap<String, Vec<String>>>,
    pub method: Option<String>,
}

#[derive(Serialize)]
pub struct Response {
    pub status: u16,
    pub headers: HashMap<String, Vec<String>>,
    pub body: String,
}

pub fn read_input() -> Result<Request, String> {
    let mut buffer = String::new();
    io::stdin().read_to_string(&mut buffer).map_err(|e| e.to_string())?;
    
    if buffer.trim().is_empty() {
        return Ok(Request { body: "".to_string(), headers: None, method: None });
    }

    serde_json::from_str(&buffer).map_err(|e| e.to_string())
}

pub fn send_response(status: u16, body: String) {
    let mut headers = HashMap::new();
    headers.insert("Content-Type".to_string(), vec!["application/json".to_string()]);
    headers.insert("X-Runtime".to_string(), vec!["Gojinn-Rust".to_string()]);

    let resp = Response { status, headers, body };
    
    if let Ok(json) = serde_json::to_string(&resp) {
        print!("{}", json);
    }
}