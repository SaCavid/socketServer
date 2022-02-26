import re, socket, select

CR_LF = "\r\n"
PHASE_START   = 1
PHASE_DATA    = 2
PHASE_CONNECT = 3


CHAR_STAR = '*'
CHAR_DOLLAR = '$'
RET_OK = '+OK'

CMD_GET = 'GET'
CMD_SET = 'SET'

class Connection(object):

    def __init__(self, fd, _fileno):
        self.data = ''
        self.fd = fd
        self.write_buffer = ''
        
        self.parser = Parser(self)
        self.fileno = _fileno

    def _consume(self, loc):
        result = self.data[:loc]
        self.data = self.data[loc:]
        return result

    def read_callback(self, chunk):
        self.data += chunk                
        loc = self.data.find(CR_LF)        
        while (loc != -1):
            self.parser.eol_callback(self._consume(loc + 2))            
            loc = self.data.find(CR_LF)


    def write(self, data):        
        self.fd.send(data)
        



class Parser(object):
    
    def __init__(self, socket):        
        self.phase = PHASE_CONNECT
        self.socket = socket
        self.buf = ''
        
    def send(self, s):        
        self.socket.write(s)
        self.socket.write(CR_LF)
    
    def sendok(self):        
        self.send(RET_OK)  

    def read(self):
        data = self.socket.readline()
        self.eol_callback(data)

    def parse_commands(self, args):
        # print "cmd args", args
        self.sendok()

    def parse_connect_line(self, line):
        #print "parse_connect_line", line
        k = line[0]
        v = line[1:].split('\r\n')[0]
        #print ">>>", k, v
        if k == CHAR_STAR:
            self.phase = PHASE_START
            self.args = {}
            self.received_arg_length = 0
            self.num_args = int(v)
            self.buf = ''
        else:
            raise Exception('commands out of order')

    def parse_start_line(self, line):        
        #print "parse_start_line", line
        if line[0] == CHAR_DOLLAR:
            self.received_arg_length = self.received_arg_length + 1
            self.phase = PHASE_DATA
            self.wait_for_data_length = int(line[1:].split('\r\n')[0])
            # print ">>> expected data len", self.wait_for_data_length
        else:
            raise Exception('commands out of order - 1')
    
    def parse_data_line(self,  line):
        #print "parse_data_line", line
        self.buf = self.buf + line
        #print "parse_data_line", "buf: (", len(self.buf), ")", self.buf

        if len(self.buf)-2 == self.wait_for_data_length:
            # print "parse_data_line all buffered...."
            self.args[self.received_arg_length] = self.buf[:-2]
            self.phase = PHASE_START
            self.buf = ''

            # did we receive everything ???
            if self.received_arg_length == self.num_args:
                self.parse_commands(self.args)
                self.args = {}
                self.phase = PHASE_CONNECT
                self.received_arg_length = 0
                self.num_args = 0

    def eol_callback(self, line):
        # line = data[0:len(data)-2] # get rid of the last crlf
        # print "phase", self.phase
        if self.phase == PHASE_CONNECT:
            self.parse_connect_line(line)           
        elif self.phase == PHASE_START:
            self.parse_start_line(line)
        elif self.phase == PHASE_DATA:
            self.parse_data_line(line)
        else:
            raise Exception('parser error')

if __name__ == "__main__":

	record = {}
	data = ""
	buffer = 60
	PORT = 5000

	server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
	server_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
	# server_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEPORT, 1) 
	server_socket.setsockopt(socket.SOL_SOCKET, socket.SO_KEEPALIVE, 1)
	server_socket.setsockopt(socket.SOL_TCP, socket.TCP_KEEPIDLE, 5)
	server_socket.setsockopt(socket.SOL_TCP, socket.TCP_KEEPINTVL, 1)
	server_socket.setsockopt(socket.SOL_TCP, socket.TCP_KEEPCNT, 8)
	server_socket.setblocking(0)
	server_socket.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
	
	server_socket.bind(("0.0.0.0", PORT))
	server_socket.listen(5000)

	READ_ONLY = ( select.POLLIN |
				  select.POLLPRI |
				  select.POLLHUP |
				  select.POLLERR )
	READ_WRITE = READ_ONLY | select.POLLOUT	
	
	poller = select.epoll()
	poller.register(server_socket, READ_ONLY)	
	
	fd_to_socket = { server_socket.fileno(): server_socket, }
	
	print("Server started on port " + str(PORT))
	
	while True:

		events = poller.poll(0.2)
		for fd, flag in events:	
			sock = fd_to_socket[fd]
			if flag & (select.POLLIN | select.POLLPRI):
				if sock == server_socket:
				
					sockfd, addr = server_socket.accept()
					fd_to_socket[ sockfd.fileno() ] = Connection(sockfd, sockfd.fileno())
					poller.register(sockfd, READ_ONLY)

				else:

					try:
						data = sock.fd.recv(buffer)
					except:
						poller.unregister(sock.fileno)
						sock.fd.close()
						continue

					if not data or len(data) == 0 or data == '' or data == -1:
					
						# Client disconnected, stop monitoring
						#print('Status: Client disconnected (Empty)')
						if sock in record:
							del record[sock]
						poller.unregister(sock.fileno)
						sock.fd.close()
						continue
						
					else:
					
						record[sock] = ""
						try:
							poller.modify(sock.fileno, READ_WRITE)
						except:
							pass
						
						data = data.decode('utf-8','ignore').encode("utf-8")
						data = str(data, 'utf-8').splitlines()
						data = data[0]
						acc = data.split('.')
						wor = data.split(acc[0])
						data = acc[0] + '' + wor[1]
						#print("INCOMING: " + data)
						
						# Disconnect duplicate
						if data in record.values():
						
							try:
								sock.fd.send("RESERVED\n".encode())
							except:
								pass
								
							#print('Status: Client disconnected (Duplicate)')
							del record[sock]
							poller.unregister(sock.fileno)
							sock.fd.close()
							continue
							
						else:
						
							# **************************
							#
							#  Receive remote command from a BOT
							#
							# **************************
							
							if "|" in data:
							
								#print('Status: Bot connected')
								host, port = sock.fd.getpeername()
								
								# Whitelisted IPs /  hosts for a BOT
								if host == "127.0.0.1" or host == "localhost" or host == "10.10.10.10":

									# Must contain exactly 1 "." and one "|"
									dots = data.count('.')
									delimeter = data.count('|')

									if dots > 1 or dots < 1 or delimeter > 1 or delimeter < 1:

										# Report error
										#print('Error: Bot -> Invalid command #1 | Data: ' + data)
										sock.fd.send("INVALID WORKER COMMAND\n".encode())
										del record[sock]
										poller.unregister(sock.fileno)
										sock.fd.close()
										continue

									else:

										dataArray = data.split('|')

										# Check for valid access key and worker name
										worker = dataArray[0]
										if re.fullmatch("[A-Za-z0-9-_.]+", worker) is None:

											#print('Error: Bot -> Invalid worker #1 | Worker: ' + worker)
											sock.fd.send("INVALID WORKER COMMAND\n".encode())
											del record[sock]
											poller.unregister(sock.fileno)
											sock.fd.close()
											continue

										else:

											# Check for valid command
											command = dataArray[1]
											if re.fullmatch("[A-Z0-9]+", command) is None:
											
												#print('Error: Bot -> Invalid command #2 | Command: ' + command)
												sock.fd.send("INVALID WORKER COMMAND\n".encode())
												del record[sock]
												poller.unregister(sock.fileno)
												sock.fd.close()
												continue

											else:

												# Check for commands
												commands = ["CONSOLE", "REBOOT", "FORCEREBOOT", "SHUTDOWN", "SAFESHUTDOWN", "INSTANTOC", "SETFANS","DOWNLOADWATTS", "RESTARTWATTS", "RESTART", "DIAG", "START", "NODERESTART", "RESTARTNODE", "STOP", "FLASH", "GETBIOS", "POWERCYCLE", "DIRECT"]
												if command not in commands:

													#print('Error: Bot -> Invalid command #3 | Command: ' + command)
													sock.fd.send("MALFORMED COMMAND\n".encode())
													del record[sock]
													poller.unregister(sock.fileno)
													sock.fd.close()
													continue

												else:
												
													# Make sure that it does not exceed the lenght
													if len(command) > 30 or len(worker) > 30:

														#print('Error: Bot -> Invalid command #4 | Command: ' + command)
														sock.fd.send("INVALID WORKER COMMAND\n".encode())
														del record[sock]
														poller.unregister(sock.fileno)
														sock.fd.close()
														continue

													else:

														# Find worker to send command to
														socketId = 0
														for s in record:
															if record[s] == worker:
																socketId = s
																break

														if socketId != 0:

															# Worker found, send command
															try:
	
																socketId.fd.send((command + "\n").encode())

															except:

																#print('Error: Bot -> Command failed | Data: ' + dataArray[0] + '|' + dataArray[1])
																sock.fd.send("FAILED\n".encode())
																del record[sock]
																del record[socketId]
																poller.unregister(sock.fileno)
																
																try:
																	poller.unregister(socketId.fileno)
																except:
																	pass
																	
																socketId.fd.close()
																sock.fd.close()
																continue

														else:

															#print('Error: Bot -> Failed search | Data: ' + dataArray[0] + '|' + dataArray[1])
															sock.fd.send("FAILED\n".encode())
															del record[sock]
															poller.unregister(sock.fileno)
															sock.fd.close()
															continue

														# Command sent successfully
														#print('Success: Bot -> Success | Data: ' + dataArray[0] + '|' + dataArray[1])
														sock.fd.send("OK\n".encode())
														del record[sock]
														poller.unregister(sock.fileno)
														sock.fd.close()
														continue

								else:
									print('Error: Bot -> Unauthorized | Hostname: ' + host)
									sock.fd.send("UNAUTHORIZED\n".encode())
									del record[sock]
									poller.unregister(sock.fileno)
									sock.fd.close()
									continue

								#print('Status: Bot disconnected')
								del record[sock]
								poller.unregister(sock.fileno)
								sock.fd.close()
								continue
								
							else:
							
								# ***********************
								#
								#  Authenticate worker
								#
								# ***********************

								if "." in data:
								
									# Must contain exactly 1 "."
									dots = data.count('.')
									if dots > 1 or dots < 1 or len(data) > 30:

										sock.fd.send("INVALID WORKER\n".encode())
										#print('Error: User -> Invalid worker | Data: ' + data)
										del record[sock]
										poller.unregister(sock.fileno)
										sock.fd.close()
										continue

									else:

										# Check for valid access key and worker name
										if re.fullmatch("[A-Za-z0-9-_.]+", data) != None:

											# Welcome new worker
											try:
												#print('Status: Worker connected ' + data)
												sock.fd.send("WELCOME\n".encode())
												record[sock] = data
											except:
												#print('Status: Timedout ' + data)
												poller.unregister(sock.fileno)
												sock.fd.close()
												continue
									
										else:

											# Report error
											try:
												sock.fd.send("INVALID WORKER\n".encode())
												#print('Error: User -> Invalid worker 2 | Data: ' + data)
												poller.unregister(sock.fileno)
												sock.fd.close()
												continue	
											except:
												#print('Error: User -> Invalid worker 2 | Data: ' + data)
												poller.unregister(sock.fileno)
												sock.fd.close()
												continue	
											
								else:

									# Report error
									try:
										sock.fd.send("INVALID WORKER\n".encode())
										#print('Error: User -> Invalid worker 2 | Data: ' + data)
										poller.unregister(sock.fileno)
										sock.fd.close()
										continue	
									except:
										#print('Error: User -> Invalid worker 2 | Data: ' + data)
										poller.unregister(sock.fileno)
										sock.fd.close()
										continue	
						
			elif flag & select.POLLHUP:
				#print('Status: Client disconnected (Empty 1)')
				del record[sock]
				poller.unregister(sock.fileno)
				sock.fd.close()

			elif flag & select.POLLERR:
				#print('Status: Client disconnected (Empty 2)')
				del record[sock]
				poller.unregister(sock.fileno)
				sock.fd.close()
			
	#print('Status: Client disconnected #3')
	sock.close()
	server_socket.close()
