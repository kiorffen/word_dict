new Vue({
    el: '#app',
    data: {
        isLoggedIn: false,
        token: '',
        words: [],
        newWord: {
            word: '',
            phonetic: '',
            definition: '',
            audioURL: ''
        },
        editingWord: null,
        loginForm: {
            username: '',
            password: ''
        },
        showChangePasswordForm: false,
        changePasswordForm: {
            oldPassword: '',
            newPassword: ''
        }
    },
    created() {
        const token = localStorage.getItem('token');
        if (token) {
            this.token = token;
            this.isLoggedIn = true;
            this.fetchWords();
        }
    },
    methods: {
        async login() {
            try {
                const response = await fetch('/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(this.loginForm)
                });

                if (!response.ok) {
                    throw new Error('Invalid credentials');
                }

                const data = await response.json();
                this.token = data.token;
                localStorage.setItem('token', data.token);
                this.isLoggedIn = true;
                this.fetchWords();
            } catch (error) {
                alert('Login failed: ' + error.message);
            }
        },

        logout() {
            this.isLoggedIn = false;
            this.token = '';
            localStorage.removeItem('token');
            this.words = [];
        },

        async changePassword() {
            try {
                const response = await fetch('/change-password', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': this.token
                    },
                    body: JSON.stringify(this.changePasswordForm)
                });

                if (!response.ok) {
                    throw new Error('Failed to change password');
                }

                this.showChangePasswordForm = false;
                this.changePasswordForm.oldPassword = '';
                this.changePasswordForm.newPassword = '';
                alert('Password changed successfully');
            } catch (error) {
                alert('Failed to change password: ' + error.message);
            }
        },

        async fetchWords() {
            try {
                const response = await fetch('/words', {
                    headers: {
                        'Authorization': this.token
                    }
                });
                const data = await response.json();
                // 确保我们正确处理返回的数据
                this.words = data.map(word => ({
                    ID: word.ID,
                    word: word.word,
                    phonetic: word.phonetic,
                    definition: word.definition,
                    audioURL: word.audioURL
                }));
            } catch (error) {
                console.error('Error fetching words:', error);
            }
        },

        async addWord() {
            try {
                const response = await fetch('/words', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': this.token
                    },
                    body: JSON.stringify(this.newWord)
                });

                if (!response.ok) {
                    throw new Error('Failed to add word');
                }

                const word = await response.json();
                // Make sure we're using the correct property names
                this.words.push({
                    ID: word.ID,
                    word: word.word,
                    phonetic: word.phonetic,
                    definition: word.definition,
                    audioURL: word.audioURL
                });
                this.newWord = {
                    word: '',
                    phonetic: '',
                    definition: '',
                    audioURL: ''
                };
            } catch (error) {
                alert('Failed to add word: ' + error.message);
            }
        },

        startEdit(word) {
            this.editingWord = { ...word };
        },

        async saveEdit() {
            try {
                const response = await fetch(`/words/${this.editingWord.ID}`, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': this.token
                    },
                    body: JSON.stringify(this.editingWord)
                });

                if (!response.ok) {
                    throw new Error('Failed to update word');
                }

                const updatedWord = await response.json();
                const index = this.words.findIndex(w => w.ID === updatedWord.ID);
                this.words.splice(index, 1, updatedWord);
                this.editingWord = null;
            } catch (error) {
                alert('Failed to update word: ' + error.message);
            }
        },

        cancelEdit() {
            this.editingWord = null;
        },

        async deleteWord(id) {
            if (!confirm('Are you sure you want to delete this word?')) {
                return;
            }

            try {
                const response = await fetch(`/words/${id}`, {
                    method: 'DELETE',
                    headers: {
                        'Authorization': this.token
                    }
                });

                if (!response.ok) {
                    throw new Error('Failed to delete word');
                }

                this.words = this.words.filter(w => w.ID !== id);
            } catch (error) {
                alert('Failed to delete word: ' + error.message);
            }
        },

        playAudio(word) {
            if (word.audioURL) {
                const audio = new Audio(word.audioURL);
                
                // 添加加载状态
                const playButton = event.currentTarget;
                const originalText = playButton.innerHTML;
                playButton.innerHTML = '🔄';
                playButton.style.pointerEvents = 'none';

                // 音频加载完成时的处理
                audio.oncanplaythrough = () => {
                    playButton.innerHTML = originalText;
                    playButton.style.pointerEvents = 'auto';
                    audio.play().catch(error => {
                        console.error('Error playing audio:', error);
                        // 如果播放失败，尝试使用备用TTS
                        const backupTTS = new Audio(`https://translate.google.com/translate_tts?ie=UTF-8&q=${word.word}&tl=en&client=tw-ob`);
                        backupTTS.play().catch(e => {
                            console.error('Backup TTS also failed:', e);
                            alert('无法播放音频');
                        });
                    });
                };

                // 音频加载失败时的处理
                audio.onerror = () => {
                    console.error('Error loading audio');
                    playButton.innerHTML = originalText;
                    playButton.style.pointerEvents = 'auto';
                    // 使用备用TTS
                    const backupTTS = new Audio(`https://translate.google.com/translate_tts?ie=UTF-8&q=${word.word}&tl=en&client=tw-ob`);
                    backupTTS.play().catch(e => {
                        console.error('Backup TTS also failed:', e);
                        alert('无法播放音频');
                    });
                };
            }
        },

        handleWordInput() {
            // 当用户输入单词时，清空之前的音标和音频URL
            this.newWord.phonetic = '';
            this.newWord.audioURL = '';
        }
    }
});
