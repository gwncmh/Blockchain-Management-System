export namespace ed25519 {
	
	export class VerificationResult {
	    Index: number;
	    PublicKey: number[];
	    Info: number[];
	    Valid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new VerificationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Index = source["Index"];
	        this.PublicKey = source["PublicKey"];
	        this.Info = source["Info"];
	        this.Valid = source["Valid"];
	    }
	}

}

export namespace main {
	
	export class BlockchainProofInfo {
	    exists: boolean;
	    signer: string;
	    timestamp: number;
	    tx_hash: string;
	    block_num: number;
	
	    static createFrom(source: any = {}) {
	        return new BlockchainProofInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exists = source["exists"];
	        this.signer = source["signer"];
	        this.timestamp = source["timestamp"];
	        this.tx_hash = source["tx_hash"];
	        this.block_num = source["block_num"];
	    }
	}

}

