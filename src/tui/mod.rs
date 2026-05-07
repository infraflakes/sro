use ratatui::{
    backend::CrosstermBackend,
    layout::Rect,
    style::{Color, Modifier, Style},
    text::{Line, Span},
    widgets::{Block, Borders, Paragraph},
    Frame, Terminal,
};
use std::sync::{Arc, Mutex};
use std::time::Duration;
use tokio::sync::broadcast;
use crossterm::{
    event::{self, DisableMouseCapture, EnableMouseCapture, Event, KeyCode},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use std::io;

const SPINNER_FRAMES: &[char] = &['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'];
const HEADER_HEIGHT: usize = 3;
const FOOTER_HEIGHT: usize = 2;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TaskStatus {
    Pending,
    Running,
    Success,
    Error,
}

#[derive(Debug, Clone)]
pub struct Task {
    pub name: String,
    pub status: TaskStatus,
    pub output: Vec<String>,
    pub expanded: bool,
}

impl Task {
    pub fn rendered_height(&self) -> usize {
        let mut height = 1; // task row
        if self.expanded && !self.output.is_empty() {
            height += 11; // panel (10) + spacing (1)
        }
        height
    }
}

#[derive(Debug, Clone)]
pub struct Model {
    pub model_type: String,
    pub name: String,
    pub tasks: Vec<Task>,
    pub selected: usize,
    pub scroll_row: usize, // Scroll offset in rows, not task indices
}

impl Model {
    pub fn new(model_type: String, name: String) -> Self {
        Self {
            model_type,
            name,
            tasks: Vec::new(),
            selected: 0,
            scroll_row: 0,
        }
    }

    pub fn add_task(&mut self, name: String) {
        self.tasks.push(Task {
            name,
            status: TaskStatus::Pending,
            output: Vec::new(),
            expanded: false,
        });
    }

    pub fn update_task_status(&mut self, index: usize, status: TaskStatus) {
        if let Some(task) = self.tasks.get_mut(index) {
            task.status = status;
        }
    }

    pub fn append_output(&mut self, idx: usize, line: String) {
        if idx < self.tasks.len() {
            self.tasks[idx].output.push(line);
        }
    }

    pub fn get_scroll_position(&self, terminal_height: u16) -> usize {
        let max_y = terminal_height.saturating_sub(HEADER_HEIGHT as u16 + FOOTER_HEIGHT as u16) as usize;
        let mut y = 0; // Start after header
        let mut visible_count = 0;
        
        for (_i, task) in self.tasks.iter().enumerate() {
            if y >= self.scroll_row {
                y += task.rendered_height();
                visible_count += 1;
                if y >= max_y + self.scroll_row {
                    break;
                }
            }
        }
        visible_count
    }

    pub fn task_y_position(&self, index: usize) -> usize {
        let mut y = 0;
        for (i, task) in self.tasks.iter().enumerate() {
            if i == index {
                break;
            }
            y += task.rendered_height();
        }
        y
    }

    pub fn scroll_to_visible(&mut self, terminal_height: u16) {
        let max_y = terminal_height.saturating_sub(HEADER_HEIGHT as u16 + FOOTER_HEIGHT as u16) as usize;
        let task_y = self.task_y_position(self.selected);
        let task_height = self.tasks[self.selected].rendered_height();
        
        // If task is above visible area, scroll up
        if task_y < self.scroll_row {
            self.scroll_row = task_y;
        }
        // If task bottom is below visible area, scroll down
        else if task_y + task_height > self.scroll_row + max_y {
            self.scroll_row = task_y + task_height - max_y;
        }
    }
}

pub struct TuiApp {
    model: Arc<Mutex<Model>>,
    tx: broadcast::Sender<TuiEvent>,
}

#[derive(Debug, Clone)]
pub enum TuiEvent {
    AddTask(String),
    UpdateStatus(usize, TaskStatus),
    AppendOutput(usize, String),
    Quit,
}

impl TuiApp {
    pub fn new(model: Model) -> Self {
        let model = Arc::new(Mutex::new(model));
        let (tx, _rx) = broadcast::channel(100);
        Self { model, tx }
    }

    pub fn get_sender(&self) -> broadcast::Sender<TuiEvent> {
        self.tx.clone()
    }

    pub fn get_model(&self) -> Arc<Mutex<Model>> {
        self.model.clone()
    }

    pub async fn run(&self) -> Result<(), io::Error> {
        enable_raw_mode()?;
        let mut stdout = io::stdout();
        execute!(stdout, EnterAlternateScreen, EnableMouseCapture)?;
        let backend = CrosstermBackend::new(stdout);
        let mut terminal = Terminal::new(backend)?;
        terminal.clear()?;

        let mut rx = self.tx.subscribe();
        let mut spinner_idx = 0;

        loop {
            // Handle events
            if let Ok(event) = rx.try_recv() {
                match event {
                    TuiEvent::Quit => break,
                    TuiEvent::AddTask(name) => {
                        let mut model = self.model.lock().unwrap();
                        model.add_task(name);
                    }
                    TuiEvent::UpdateStatus(idx, status) => {
                        let mut model = self.model.lock().unwrap();
                        model.update_task_status(idx, status);
                    }
                    TuiEvent::AppendOutput(idx, line) => {
                        let mut model = self.model.lock().unwrap();
                        model.append_output(idx, line);
                    }
                }
            }

            // Handle keyboard events
            if event::poll(Duration::from_millis(50))?
                && let Event::Key(key) = event::read()? {
                    match key.code {
                        KeyCode::Char('q') => break,
                        KeyCode::Down => {
                            let mut model = self.model.lock().unwrap();
                            if model.selected < model.tasks.len().saturating_sub(1) {
                                model.selected += 1;
                                let size = terminal.size()?;
                                model.scroll_to_visible(size.height);
                            }
                        }
                        KeyCode::Up => {
                            let mut model = self.model.lock().unwrap();
                            if model.selected > 0 {
                                model.selected -= 1;
                                let size = terminal.size()?;
                                model.scroll_to_visible(size.height);
                            }
                        }
                        KeyCode::Enter | KeyCode::Char(' ') | KeyCode::Right => {
                            let selected = {
                                let model = self.model.lock().unwrap();
                                model.selected
                            };
                            let mut model = self.model.lock().unwrap();
                            if let Some(task) = model.tasks.get_mut(selected) {
                                task.expanded = !task.expanded;
                                let size = terminal.size()?;
                                model.scroll_to_visible(size.height);
                            }
                        }
                        KeyCode::Left => {
                            let selected = {
                                let model = self.model.lock().unwrap();
                                model.selected
                            };
                            let mut model = self.model.lock().unwrap();
                            if let Some(task) = model.tasks.get_mut(selected) {
                                if task.expanded {
                                    task.expanded = false;
                                }
                            }
                        }
                        _ => {}
                    }
                }

            // Render
            spinner_idx = (spinner_idx + 1) % SPINNER_FRAMES.len();
            terminal.draw(|f| {
                let model = self.model.lock().unwrap();
                render(f, &model, spinner_idx);
            })?;
        }

        disable_raw_mode()?;
        execute!(
            terminal.backend_mut(),
            LeaveAlternateScreen,
            DisableMouseCapture
        )?;
        Ok(())
    }
}

fn render(f: &mut Frame, model: &Model, spinner_idx: usize) {
    let size = f.size();

    // Header with badge and info
    let badge_color = match model.model_type.as_str() {
        "seq" => Color::Blue,
        "par" => Color::Magenta,
        "sync" => Color::Cyan,
        _ => Color::White,
    };

    let header_spans = vec![
        Span::styled(
            format!(" {} ", model.model_type),
            Style::default().fg(Color::Black).bg(badge_color),
        ),
        Span::styled(
            format!(" {} ", model.name),
            Style::default().fg(Color::White),
        ),
        Span::styled(
            format!(" {} tasks ", model.tasks.len()),
            Style::default().fg(Color::DarkGray),
        ),
    ];

    let header = Paragraph::new(Line::from(header_spans));
    f.render_widget(header, Rect::new(0, 0, size.width, 1));

    // Separator
    let separator = Paragraph::new("─".repeat(size.width as usize))
        .style(Style::default().fg(Color::DarkGray));
    f.render_widget(separator, Rect::new(0, 1, size.width, 1));

    // Task rows with accordion expansion
    let mut y = 0;
    let max_y = size.height.saturating_sub(HEADER_HEIGHT as u16 + FOOTER_HEIGHT as u16) as usize;

    for (i, task) in model.tasks.iter().enumerate() {
        let task_row = y;
        if task_row >= model.scroll_row {
            if y - model.scroll_row >= max_y {
                break;
            }

            // Task row
            let spinner = if task.status == TaskStatus::Running {
                SPINNER_FRAMES[spinner_idx].to_string()
            } else {
                match task.status {
                    TaskStatus::Pending => "·".to_string(),
                    TaskStatus::Success => "✓".to_string(),
                    TaskStatus::Error => "✗".to_string(),
                    TaskStatus::Running => unreachable!(),
                }
            };

            let color = match task.status {
                TaskStatus::Pending => Color::DarkGray,
                TaskStatus::Running => Color::Yellow,
                TaskStatus::Success => Color::Green,
                TaskStatus::Error => Color::Red,
            };

            let is_selected = i == model.selected;
            let style = if is_selected {
                Style::default().fg(color).add_modifier(Modifier::BOLD)
            } else {
                Style::default().fg(color)
            };

            let arrow = if task.status != TaskStatus::Pending {
                if task.expanded { "▼" } else { "▶" }
            } else {
                ""
            };

            let task_line = format!("{} {} {} {}",
                if is_selected { "▸" } else { " " },
                spinner,
                task.name,
                arrow
            );

            let task_paragraph = Paragraph::new(task_line)
                .style(style);
            f.render_widget(task_paragraph, Rect::new(0, (y - model.scroll_row) as u16 + HEADER_HEIGHT as u16, size.width, 1));
            y += 1;

            // Expanded output panel - fixed height to keep layout static
            if task.expanded {
                const PANEL_HEIGHT: usize = 10;
                let panel_height = PANEL_HEIGHT.min(max_y - (y - model.scroll_row));
                
                if panel_height > 0 && !task.output.is_empty() {
                    let output_lines: Vec<String> = task.output
                        .iter()
                        .rev()
                        .take(panel_height)
                        .cloned()
                        .collect();

                    let output_text: Vec<Line> = output_lines
                        .iter()
                        .rev()
                        .map(|line| Line::from(line.as_str()))
                        .collect();

                    let output_paragraph = Paragraph::new(output_text)
                        .style(Style::default().fg(Color::Gray))
                        .block(Block::default().borders(Borders::LEFT).border_style(Style::default().fg(Color::DarkGray)));
                    f.render_widget(output_paragraph, Rect::new(2, (y - model.scroll_row) as u16 + HEADER_HEIGHT as u16, size.width - 4, panel_height as u16));
                    y += panel_height;
                }
                y += 1; // spacing after panel
            }
        } else {
            y += task.rendered_height();
        }
    }

    // Footer with counts
    let mut ok_count = 0;
    let mut running_count = 0;
    let mut pending_count = 0;
    let mut error_count = 0;

    for task in &model.tasks {
        match task.status {
            TaskStatus::Success => ok_count += 1,
            TaskStatus::Running => running_count += 1,
            TaskStatus::Pending => pending_count += 1,
            TaskStatus::Error => error_count += 1,
        }
    }

    let mut footer_spans = Vec::new();

    if ok_count > 0 {
        footer_spans.push(Span::styled(format!("✓ {} ok ", ok_count), Style::default().fg(Color::Green)));
    }
    if running_count > 0 {
        footer_spans.push(Span::styled(format!("{} {} running ", SPINNER_FRAMES[spinner_idx], running_count), Style::default().fg(Color::Yellow)));
    }
    if pending_count > 0 {
        footer_spans.push(Span::styled(format!("· {} pending ", pending_count), Style::default().fg(Color::DarkGray)));
    }
    if error_count > 0 {
        footer_spans.push(Span::styled(format!("✗ {} error ", error_count), Style::default().fg(Color::Red)));
    }

    let footer = Paragraph::new(Line::from(footer_spans));
    f.render_widget(footer, Rect::new(1, size.height - 1, size.width - 2, 1));

    // Footer separator
    let footer_sep = Paragraph::new("─".repeat(size.width as usize))
        .style(Style::default().fg(Color::DarkGray));
    f.render_widget(footer_sep, Rect::new(0, size.height - 2, size.width, 1));
}
