import grpc
import holloman_pb2
import holloman_pb2_grpc
import binascii

def read_file(file_path):
    with open(file_path, 'rb') as f:
        return f.read()

def run(file_path, server_address='localhost:50051'):
    # Create a secure channel
    channel = grpc.insecure_channel(server_address)
    
    try:
        # Create a stub (client)
        stub = holloman_pb2_grpc.HollomanStub(channel)

        # First, check service capabilities
        capabilities = holloman_pb2.ServiceCapabilities(
            Acceleration="default",
            MaxOrder=1
        )
        
        service_caps = stub.Capabilities(capabilities)
        print(f"Service capabilities received:")
        print(f"Acceleration: {service_caps.Acceleration}")
        print(f"MaxOrder: {service_caps.MaxOrder}")

        # Read the file
        file_content = read_file(file_path)
        
        # Create buffer request
        request = holloman_pb2.BufferRequest(
            Buffer=file_content
        )

        # Call ClusterBuffer
        response = stub.ClusterBuffer(request)
        
        print("\nBuffer Response received:")
        print(f"HOrder: {response.HOrder}")
        print(f"Identifier: {response.Identifer}")
        print(f"Magic: {response.Magic}")

        print("%s" %( binascii.hexlify(response.Identifer).decode()))

    except grpc.RpcError as e:
        print(f"RPC failed: {e.code()}")
        print(f"Details: {e.details()}")
    
    finally:
        channel.close()

if __name__ == '__main__':
    import sys
    
    if len(sys.argv) < 2:
        print("Usage: python client.py <file_path> [server_address]")
        sys.exit(1)
    
    file_path = sys.argv[1]
    server_address = sys.argv[2] if len(sys.argv) > 2 else 'localhost:50051'
    
    run(file_path, server_address)

