function eventManager() {
    return {
        events: [],
        filteredEvents: [],
        groupedEvents: [],
        stats: {},
        loading: false,
        statusFilter: '',
        hideSingleEvents: false,

        async loadEvents() {
            this.loading = true;
            try {
                const response = await fetch('/api/events');
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                this.events = await response.json();
                this.filterEvents();
            } catch (error) {
                console.error('Error loading events:', error);
                alert('Failed to load events. Please try again.');
            } finally {
                this.loading = false;
            }
        },

        async loadStats() {
            try {
                const response = await fetch('/api/events/stats');
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                this.stats = await response.json();
            } catch (error) {
                console.error('Error loading stats:', error);
            }
        },

        groupEventsByDate(events) {
            const eventsByDate = {};

            for (const event of events) {
                const date = new Date(event.start_time).toISOString().split('T')[0];
                if (!eventsByDate[date]) {
                    eventsByDate[date] = [];
                }
                eventsByDate[date].push(event);
            }

            // Convert to array and sort by date proximity to current date
            const result = [];
            for (const [date, dateEvents] of Object.entries(eventsByDate)) {
                result.push({
                    date: date,
                    events: dateEvents
                });
            }

            // Sort by date proximity (closest to current date first)
            const nowDate = new Date().toISOString().split('T')[0];
            result.sort((a, b) => {
                const dateA = new Date(a.date);
                const dateB = new Date(b.date);
                const now = new Date(nowDate);

                const diffA = Math.abs((dateA - now) / (1000 * 60 * 60 * 24));
                const diffB = Math.abs((dateB - now) / (1000 * 60 * 60 * 24));

                return diffA - diffB;
            });

            return result;
        },

        filterEvents() {
            let filtered = this.events;

            // Apply status filter
            if (this.statusFilter === 'approved') {
                filtered = filtered.filter(event => !event.rejected);
            } else if (this.statusFilter === 'rejected') {
                filtered = filtered.filter(event => event.rejected);
            }

            // Group events by date
            this.groupedEvents = this.groupEventsByDate(filtered);

            // Apply single event filter
            if (this.hideSingleEvents) {
                this.groupedEvents = this.groupedEvents.filter(dateGroup => dateGroup.events.length > 1);
            }

            this.filteredEvents = this.groupedEvents;
        },

        async updateEventStatus(uid, approved) {
            const rejected = !approved;

            try {
                const response = await fetch(`/api/events/${uid}/status`, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ rejected: rejected })
                });

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                // Update the event in our local data
                this.events.forEach(event => {
                    if (event.uid === uid) {
                        event.rejected = rejected;
                    }
                });

                // Re-filter and reload stats
                this.filterEvents();
                this.loadStats();

                // Show success message
                this.showNotification('Event status updated successfully!', 'success');
            } catch (error) {
                console.error('Error updating status:', error);
                this.showNotification('Failed to update event status. Please try again.', 'error');
            }
        },

        showNotification(message, type = 'info') {
            // Create notification element
            const notification = document.createElement('div');
            notification.className = `fixed top-4 right-4 px-6 py-3 rounded-lg shadow-lg z-50 transition-all duration-300 ${
                type === 'success' ? 'bg-green-500 text-white' :
                type === 'error' ? 'bg-red-500 text-white' :
                'bg-blue-500 text-white'
            }`;
            notification.textContent = message;

            // Add to page
            document.body.appendChild(notification);

            // Remove after 3 seconds
            setTimeout(() => {
                notification.style.opacity = '0';
                setTimeout(() => {
                    if (notification.parentNode) {
                        notification.parentNode.removeChild(notification);
                    }
                }, 300);
            }, 3000);
        },

        formatDate(dateString) {
            const parts = dateString.split("-");
            const date = new Date(parseInt(parts[0]), parseInt(parts[1]) - 1, parseInt(parts[2]));
            const today = new Date();
            const tomorrow = new Date(today);
            tomorrow.setDate(tomorrow.getDate() + 1);

            if (date.toDateString() === today.toDateString()) {
                return 'Today';
            } else if (date.toDateString() === tomorrow.toDateString()) {
                return 'Tomorrow';
            } else {
                return date.toLocaleDateString('en-US', {
                    weekday: 'long',
                    year: 'numeric',
                    month: 'long',
                    day: 'numeric'
                });
            }
        },

        formatTime(timeString) {
            const date = new Date(timeString);
            return date.toLocaleTimeString('en-US', {
                hour: 'numeric',
                minute: '2-digit',
                hour12: true
            });
        },



        get totalEvents() {
            return this.events.length;
        }
    };
}