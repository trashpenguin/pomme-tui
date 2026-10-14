use crate::protocol::WireTrack;

#[derive(Default)]
pub struct Queue {
    pub items: Vec<WireTrack>,
    pub pos: usize,
    pub repeat: u8,
    pub shuffle: bool,
    history: Vec<usize>,
    remaining: Vec<usize>,
}
impl Queue {
    pub fn new(items: Vec<WireTrack>) -> Self {
        Self {
            items,
            ..Default::default()
        }
    }
    /// Selects the next item; repeat-one only affects automatic completion.
    pub fn advance(&mut self, explicit: bool) -> Option<usize> {
        if self.items.is_empty() {
            return None;
        }
        if self.repeat == 1 && !explicit {
            return Some(self.pos);
        }
        let next = if self.shuffle {
            if self.remaining.is_empty() && self.repeat == 2 {
                self.refill();
            }
            self.remaining
                .first()
                .copied()
                .or_else(|| (self.repeat == 2).then_some(self.pos))
        } else if self.pos + 1 < self.items.len() {
            Some(self.pos + 1)
        } else {
            (self.repeat == 2).then_some(0)
        }?;
        if self.shuffle && !self.remaining.is_empty() {
            self.remaining.remove(0);
        }
        self.history.push(self.pos);
        self.pos = next;
        Some(next)
    }
    pub fn previous(&mut self) -> Option<usize> {
        let previous = self.history.pop().or_else(|| self.pos.checked_sub(1))?;
        self.pos = previous;
        Some(previous)
    }
    pub fn jump(&mut self, index: usize) -> anyhow::Result<()> {
        anyhow::ensure!(index < self.items.len(), "queue index is out of range");
        if index != self.pos {
            self.history.push(self.pos);
        }
        self.pos = index;
        self.remaining.retain(|&i| i != index);
        Ok(())
    }
    pub fn insert_next(&mut self, items: Vec<WireTrack>) {
        let count = items.len();
        let index = (self.pos + 1).min(self.items.len());
        for old in self.history.iter_mut().chain(self.remaining.iter_mut()) {
            if *old >= index {
                *old += count;
            }
        }
        self.items.splice(index..index, items);
        if self.shuffle {
            self.remaining.splice(0..0, index..index + count);
        }
    }
    pub fn append(&mut self, items: Vec<WireTrack>) {
        let start = self.items.len();
        self.items.extend(items);
        if self.shuffle {
            self.remaining.extend(start..self.items.len());
        }
    }
    pub fn cycle_repeat(&mut self) {
        self.repeat = (self.repeat + 1) % 3;
    }
    pub fn toggle_shuffle(&mut self) {
        self.shuffle = !self.shuffle;
        self.refill();
    }
    fn refill(&mut self) {
        self.remaining = (0..self.items.len()).filter(|&i| i != self.pos).collect();
        if self.shuffle {
            fastrand::shuffle(&mut self.remaining);
        }
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    fn song(id: &str) -> WireTrack {
        WireTrack {
            id: id.into(),
            title: id.into(),
            kind: "song".into(),
            dur_ms: 180000,
            ..Default::default()
        }
    }
    #[test]
    fn automatic_repeat_one_does_not_trap_explicit_next() {
        let mut q = Queue::new(vec![song("a"), song("b")]);
        q.cycle_repeat();
        assert_eq!(q.advance(false), Some(0));
        assert_eq!(q.advance(true), Some(1));
    }
    #[test]
    fn empty_single_end_and_repeat_all() {
        assert_eq!(Queue::default().advance(false), None);
        let mut q = Queue::new(vec![song("a")]);
        assert_eq!(q.advance(false), None);
        q.cycle_repeat();
        q.cycle_repeat();
        assert_eq!(q.advance(false), Some(0));
    }
    #[test]
    fn insert_append_and_invalid_jump_preserve_current() {
        let mut q = Queue::new(vec![song("a"), song("d")]);
        q.insert_next(vec![song("b")]);
        q.append(vec![song("e")]);
        assert_eq!(
            q.items.iter().map(|s| s.id.as_str()).collect::<Vec<_>>(),
            ["a", "b", "d", "e"]
        );
        assert!(q.jump(99).is_err());
        assert_eq!(q.pos, 0);
    }
    #[test]
    fn shuffle_visits_members_once_and_previous_uses_history() {
        let mut q = Queue::new((0..8).map(|i| song(&i.to_string())).collect());
        q.toggle_shuffle();
        let mut visited = vec![0];
        while let Some(i) = q.advance(true) {
            visited.push(i);
        }
        assert_eq!(visited.len(), 8);
        let last = q.pos;
        assert_eq!(q.previous(), Some(visited[6]));
        assert_ne!(q.pos, last);
        visited.sort();
        assert_eq!(visited, (0..8).collect::<Vec<_>>());
        assert_eq!(q.items.len(), 8);
    }
}
