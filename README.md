# Appointment Service

The **Appointment Service** handles the scheduling and management of doctor-patient appointments, along with additional functionalities like video calls, prescription management, and alerts.

---

## **Features**

### **Appointment Management**
- Book, update, and cancel doctor-patient appointments.
- Store appointment details securely in the database.

### **Video Call Integration**
- Enable virtual consultations via **Jitsi**.

### **Prescription Management**
- Manage and store prescriptions issued during appointments.

### **Daily Appointment Alerts**
- Send automated reminders and updates to patients using **Go Cron** for scheduling tasks.

---

## **Technology Stack**
- **Backend:** Go (Golang)
- **Task Scheduling:** Go Cron
- **Video Calls:** Jitsi Meet API
- **Database:** MongoDB or PostgreSQL (depending on setup)

---

## **How to Run**

### Clone the Repository
```bash
git clone https://github.com/NUHMANUDHEENT/hosp-connect-appointment-service.git
cd appointment-service
